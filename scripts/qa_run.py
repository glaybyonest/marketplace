from __future__ import annotations

import json
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
ARTIFACTS_DIR = ROOT / "qa-artifacts"
RAW_DIR = ARTIFACTS_DIR / "raw"
BASE_URL = "http://localhost:18080"
DB_CONTAINER = "marketplace-postgres"

ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)
RAW_DIR.mkdir(parents=True, exist_ok=True)


state: dict[str, Any] = {
    "run_id": datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S"),
    "base_url": BASE_URL,
    "customer_email": "",
    "customer_password": "Secure123!",
    "customer_new_password": "NewSecure123!",
    "customer_full_name": "QA User 001",
    "customer_tokens": None,
    "customer_tokens_2": None,
    "verify_token": None,
    "reset_token": None,
    "product": None,
    "seller_category_id": None,
    "place_crud_id": None,
    "order_place_id": None,
    "order_id": None,
    "secondary_session_id": None,
    "seller_tokens": None,
    "seller_product_id": None,
    "results": [],
}
state["customer_email"] = f"qa.codex+{state['run_id']}@example.com"


def save_text_artifact(name: str, content: str) -> str:
    path = RAW_DIR / name
    path.write_text(content, encoding="utf-8")
    return str(path)


def redact_string(value: str) -> str:
    if not isinstance(value, str) or not value:
        return value

    patterns = [
        (r'(?i)("access_token"\s*:\s*")([^"]+)(")', r'\1[REDACTED]\3'),
        (r'(?i)("refresh_token"\s*:\s*")([^"]+)(")', r'\1[REDACTED]\3'),
        (r'(?i)("token"\s*:\s*")([^"]+)(")', r'\1[REDACTED]\3'),
        (r'(?i)("password"\s*:\s*")([^"]+)(")', r'\1[REDACTED]\3'),
        (r'(?i)("new_password"\s*:\s*")([^"]+)(")', r'\1[REDACTED]\3'),
        (r"(?i)(Bearer\s+)[A-Za-z0-9._=-]+", r"\1[REDACTED]"),
        (r"(?i)(token=)[A-Za-z0-9._-]+", r"\1[REDACTED]"),
    ]

    redacted = value
    for pattern, replacement in patterns:
        redacted = re.sub(pattern, replacement, redacted)
    return redacted


def redact_value(value: Any, key: str = "") -> Any:
    lowered_key = key.lower()
    if isinstance(value, dict):
        return {item_key: redact_value(item_value, item_key) for item_key, item_value in value.items()}
    if isinstance(value, list):
        return [redact_value(item, key) for item in value]
    if isinstance(value, str):
        if lowered_key == "authorization":
            return "Bearer [REDACTED]"
        if "token" in lowered_key or "password" in lowered_key or "secret" in lowered_key:
            return "[REDACTED]"
        return redact_string(value)
    return value


def save_json_artifact(name: str, data: Any) -> str:
    return save_text_artifact(name, json.dumps(redact_value(data), ensure_ascii=False, indent=2))


def api_request(
    method: str,
    path: str,
    *,
    body: Any | None = None,
    headers: dict[str, str] | None = None,
    artifact_name: str | None = None,
) -> dict[str, Any]:
    url = path if path.startswith("http") else f"{state['base_url']}{path}"
    payload = None
    request_headers = dict(headers or {})
    if body is not None:
        payload = json.dumps(body).encode("utf-8")
        request_headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=payload, headers=request_headers, method=method.upper())

    status_code: int
    response_body = ""
    response_headers: dict[str, str] = {}
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            status_code = response.getcode()
            response_body = response.read().decode("utf-8")
            response_headers = dict(response.headers.items())
    except urllib.error.HTTPError as exc:
        status_code = exc.code
        response_body = exc.read().decode("utf-8")
        response_headers = dict(exc.headers.items())

    response_json = None
    if response_body:
        try:
            response_json = json.loads(response_body)
        except json.JSONDecodeError:
            response_json = None

    artifact = {
        "method": method.upper(),
        "path": path,
        "url": url,
        "status_code": status_code,
        "request_body": body,
        "request_headers": request_headers,
        "response_body": response_body,
        "response_json": response_json,
        "response_headers": response_headers,
    }
    if artifact_name:
        artifact["artifact_path"] = save_json_artifact(artifact_name, artifact)
    return artifact


def db_scalar(sql: str) -> str:
    completed = subprocess.run(
        [
            "docker",
            "exec",
            DB_CONTAINER,
            "sh",
            "-lc",
            f'psql -U postgres -d marketplace -t -A -c "{sql.replace(chr(34), r"\\\"")}"',
        ],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"psql failed: {sql}\nstdout={completed.stdout}\nstderr={completed.stderr}")
    return completed.stdout.strip()


def db_row(sql: str) -> dict[str, Any] | None:
    raw = db_scalar(f"SELECT COALESCE(row_to_json(q)::text, '') FROM ({sql}) q;")
    return None if not raw else json.loads(raw)


def get_latest_email_token(email: str, subject_like: str) -> dict[str, Any]:
    safe_email = email.replace("'", "''")
    safe_subject = subject_like.replace("'", "''")
    row = db_row(
        f"""
SELECT id, recipient, subject, body_text, status, created_at, sent_at
FROM email_jobs
WHERE recipient = '{safe_email}'
  AND subject ILIKE '%{safe_subject}%'
ORDER BY created_at DESC
LIMIT 1
""".strip()
    )
    if row is None:
        raise RuntimeError(f"email job not found for {email} / {subject_like}")
    match = re.search(r"token=([A-Za-z0-9_.-]+)", row["body_text"])
    if not match:
        raise RuntimeError("token not found in email body")
    return {"token": match.group(1), "email_job": row}


def find_seed_product() -> dict[str, Any]:
    row = db_row(
        """
SELECT p.id, p.name, p.slug, p.price, p.stock_qty, p.category_id
FROM products p
WHERE p.is_active = TRUE
  AND p.stock_qty >= 10
ORDER BY p.stock_qty DESC, p.created_at ASC
LIMIT 1
""".strip()
    )
    if row is None:
        raise RuntimeError("no active product with stock >= 10")
    return row


def find_seed_category() -> dict[str, Any]:
    row = db_row(
        """
SELECT id, name, slug
FROM categories
ORDER BY created_at ASC
LIMIT 1
""".strip()
    )
    if row is None:
        raise RuntimeError("no category found")
    return row


def auth_header(access_token: str) -> dict[str, str]:
    return {"Authorization": f"Bearer {access_token}"}


def login_user(email: str, password: str, artifact_name: str) -> dict[str, Any]:
    return api_request(
        "POST",
        "/api/v1/auth/login",
        body={"email": email, "password": password},
        artifact_name=artifact_name,
    )


def record_case(
    *,
    case_id: str,
    priority: str,
    name: str,
    precondition: str,
    steps: list[str],
    test_data: dict[str, Any],
    expected: str,
    actual: str,
    status: str,
    evidence: list[str],
    comment: str = "",
) -> None:
    state["results"].append(
        {
            "id": case_id,
            "priority": priority,
            "name": redact_string(name),
            "precondition": redact_string(precondition),
            "steps": [redact_string(step) for step in steps],
            "test_data": redact_value(test_data),
            "expected_result": redact_string(expected),
            "actual_result": redact_string(actual),
            "status": status,
            "evidence": [redact_string(item) for item in evidence],
            "comment": redact_string(comment),
        }
    )


def response_data(response: dict[str, Any]) -> Any:
    payload = response.get("response_json")
    if isinstance(payload, dict):
        return payload.get("data")
    return None


def response_items(response: dict[str, Any]) -> list[dict[str, Any]]:
    data = response_data(response)
    if isinstance(data, dict) and isinstance(data.get("items"), list):
        return data["items"]
    return []


def write_reports() -> tuple[str, str]:
    json_path = ARTIFACTS_DIR / "test-results.json"
    md_path = ARTIFACTS_DIR / "test-report.md"

    summary_lines = [
        f"| {result['id']} | {result['name']} | {result['actual_result'].replace(chr(10), ' ')} | {result['status']} |"
        for result in state["results"]
    ]

    detail_blocks = []
    for result in state["results"]:
        detail_blocks.append(
            "\n".join(
                [
                    f"Тестовый пример #: {result['id']}",
                    f"Приоритет: {result['priority']}",
                    f"Название: {result['name']}",
                    f"Предварительное условие: {result['precondition']}",
                    f"Шаги выполнения: {'; '.join(result['steps'])}",
                    f"Тестовые данные: {json.dumps(result['test_data'], ensure_ascii=False)}",
                    f"Ожидаемый результат: {result['expected_result']}",
                    f"Фактический результат: {result['actual_result']}",
                    f"Статус: {result['status']}",
                    f"Доказательства: {'; '.join(result['evidence'])}",
                    f"Комментарий: {result['comment']}",
                ]
            )
        )

    md = "\n".join(
        [
            "# QA Test Report",
            "",
            "- Как поднял проект: `docker compose up -d postgres`, затем локальный запуск backend через `scripts/start-qa-api.ps1`.",
            "- Какие сервисы запускал: `postgres` в Docker и backend API на `18080`.",
            f"- Какой base URL использовал: `{state['base_url']}`.",
            "- Какой auth mode использовал: bearer (`AUTH_COOKIE_MODE=false`).",
            "- Где брал email verify/reset токены: из `email_jobs.body_text` через `docker exec ... psql`.",
            "",
            "| ID | Название теста | Фактический результат | Статус |",
            "| --- | --- | --- | --- |",
            *summary_lines,
            "",
            *detail_blocks,
            "",
        ]
    )

    machine = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "base_url": state["base_url"],
        "auth_mode": "bearer",
        "token_source": "email_jobs.body_text",
        "results": state["results"],
    }
    json_path.write_text(json.dumps(machine, ensure_ascii=False, indent=2), encoding="utf-8")
    md_path.write_text(md, encoding="utf-8")
    return str(json_path), str(md_path)


def main() -> int:
    ready = api_request("GET", "/readyz", artifact_name="00-readyz.json")
    if ready["status_code"] != 200:
        raise RuntimeError(f"backend is not ready: {ready['status_code']}")

    state["product"] = find_seed_product()
    state["seller_category_id"] = state["product"]["category_id"]

    register = api_request(
        "POST",
        "/api/v1/auth/register",
        body={
            "email": state["customer_email"],
            "password": state["customer_password"],
            "full_name": state["customer_full_name"],
        },
        artifact_name="01-auth-register.json",
    )
    login_before_verify = login_user(
        state["customer_email"],
        state["customer_password"],
        "01-auth-login-before-verify.json",
    )
    auth1_ok = (
        register["status_code"] == 201
        and register["response_json"]["data"]["user"]["is_email_verified"] is False
        and register["response_json"]["data"]["requires_email_verification"] is True
        and login_before_verify["status_code"] == 403
        and not (login_before_verify["response_json"] or {}).get("data", {}).get("tokens")
    )
    record_case(
        case_id="TC_API_AUTH_001",
        priority="Высокий",
        name="Регистрация нового пользователя и запрет входа до подтверждения email",
        precondition=f"Backend доступен на {state['base_url']}, email {state['customer_email']} уникален для этого прогона.",
        steps=[
            "POST /api/v1/auth/register",
            "POST /api/v1/auth/login до подтверждения email",
        ],
        test_data={
            "email": state["customer_email"],
            "password": state["customer_password"],
            "full_name": state["customer_full_name"],
        },
        expected="Регистрация 201; requires_email_verification=true; is_email_verified=false; login до подтверждения возвращает 403 и не выдает токены.",
        actual=(
            f"register -> {register['status_code']}, "
            f"is_email_verified={register['response_json']['data']['user']['is_email_verified']}, "
            f"requires_email_verification={register['response_json']['data']['requires_email_verification']}; "
            f"login до верификации -> {login_before_verify['status_code']}, "
            f"tokens_present={bool((login_before_verify['response_json'] or {}).get('data', {}).get('tokens'))}."
        ),
        status="Зачет" if auth1_ok else "Незачет",
        evidence=[
            f"POST /api/v1/auth/register -> {register['status_code']}, артефакт: {register['artifact_path']}",
            f"POST /api/v1/auth/login -> {login_before_verify['status_code']}, артефакт: {login_before_verify['artifact_path']}",
        ],
    )

    verify_request = api_request(
        "POST",
        "/api/v1/auth/verify-email/request",
        body={"email": state["customer_email"]},
        artifact_name="02-auth-verify-request.json",
    )
    verify_info = get_latest_email_token(state["customer_email"], "verify")
    state["verify_token"] = verify_info["token"]
    save_json_artifact("02-auth-verify-email-job.json", verify_info["email_job"])
    verify_confirm = api_request(
        "POST",
        "/api/v1/auth/verify-email/confirm",
        body={"token": state["verify_token"]},
        artifact_name="02-auth-verify-confirm.json",
    )
    login_after_verify = login_user(
        state["customer_email"],
        state["customer_password"],
        "02-auth-login-after-verify.json",
    )
    state["customer_tokens"] = login_after_verify["response_json"]["data"]["tokens"]
    auth2_ok = (
        verify_request["status_code"] == 200
        and verify_confirm["status_code"] == 200
        and login_after_verify["status_code"] == 200
        and bool(state["customer_tokens"]["access_token"])
        and bool(state["customer_tokens"]["refresh_token"])
        and login_after_verify["response_json"]["data"]["requires_email_verification"] is False
    )
    record_case(
        case_id="TC_API_AUTH_002",
        priority="Высокий",
        name="Подтверждение email токеном и успешная авторизация пользователя",
        precondition="Пользователь зарегистрирован и еще не подтвержден.",
        steps=[
            "POST /api/v1/auth/verify-email/request",
            "Получение verify_token из email_jobs",
            "POST /api/v1/auth/verify-email/confirm",
            "POST /api/v1/auth/login",
        ],
        test_data={"email": state["customer_email"], "verify_token": state["verify_token"]},
        expected="Запрос новой ссылки 200; подтверждение 200; login 200; выданы access_token/refresh_token; requires_email_verification=false.",
        actual=(
            f"verify request -> {verify_request['status_code']}; "
            f"verify confirm -> {verify_confirm['status_code']}; "
            f"login -> {login_after_verify['status_code']}; "
            f"access_token_present={bool(state['customer_tokens']['access_token'])}; "
            f"refresh_token_present={bool(state['customer_tokens']['refresh_token'])}; "
            f"requires_email_verification={login_after_verify['response_json']['data']['requires_email_verification']}."
        ),
        status="Зачет" if auth2_ok else "Незачет",
        evidence=[
            f"POST /api/v1/auth/verify-email/request -> {verify_request['status_code']}, артефакт: {verify_request['artifact_path']}",
            f"email_jobs verify token={state['verify_token']}, артефакт: {RAW_DIR / '02-auth-verify-email-job.json'}",
            f"POST /api/v1/auth/verify-email/confirm -> {verify_confirm['status_code']}, артефакт: {verify_confirm['artifact_path']}",
            f"POST /api/v1/auth/login -> {login_after_verify['status_code']}, артефакт: {login_after_verify['artifact_path']}",
        ],
    )

    reset_request = api_request(
        "POST",
        "/api/v1/auth/password-reset/request",
        body={"email": state["customer_email"]},
        artifact_name="03-auth-reset-request.json",
    )
    reset_info = get_latest_email_token(state["customer_email"], "reset")
    state["reset_token"] = reset_info["token"]
    save_json_artifact("03-auth-reset-email-job.json", reset_info["email_job"])
    reset_confirm = api_request(
        "POST",
        "/api/v1/auth/password-reset/confirm",
        body={"token": state["reset_token"], "new_password": state["customer_new_password"]},
        artifact_name="03-auth-reset-confirm.json",
    )
    login_old = login_user(state["customer_email"], state["customer_password"], "03-auth-login-old-password.json")
    login_new = login_user(state["customer_email"], state["customer_new_password"], "03-auth-login-new-password.json")
    state["customer_tokens"] = login_new["response_json"]["data"]["tokens"]
    reset_reuse = api_request(
        "POST",
        "/api/v1/auth/password-reset/confirm",
        body={"token": state["reset_token"], "new_password": "AnotherSecure123!"},
        artifact_name="03-auth-reset-reuse.json",
    )
    auth3_ok = (
        reset_request["status_code"] == 200
        and reset_confirm["status_code"] == 200
        and login_old["status_code"] == 401
        and login_new["status_code"] == 200
        and reset_reuse["status_code"] in (400, 404)
    )
    record_case(
        case_id="TC_API_AUTH_003",
        priority="Высокий",
        name="Восстановление пароля через запрос и подтверждение одноразового токена",
        precondition="Email пользователя уже подтвержден.",
        steps=[
            "POST /api/v1/auth/password-reset/request",
            "Получение reset_token из email_jobs",
            "POST /api/v1/auth/password-reset/confirm",
            "POST /api/v1/auth/login старым паролем",
            "POST /api/v1/auth/login новым паролем",
            "Повторное использование reset_token",
        ],
        test_data={
            "email": state["customer_email"],
            "old_password": state["customer_password"],
            "new_password": state["customer_new_password"],
            "reset_token": state["reset_token"],
        },
        expected="request 200; confirm 200; старый пароль возвращает 401; новый пароль возвращает 200; повторное использование токена возвращает 400/404.",
        actual=(
            f"reset request -> {reset_request['status_code']}; "
            f"reset confirm -> {reset_confirm['status_code']}; "
            f"login old -> {login_old['status_code']}; "
            f"login new -> {login_new['status_code']}; "
            f"reset reuse -> {reset_reuse['status_code']}."
        ),
        status="Зачет" if auth3_ok else "Незачет",
        evidence=[
            f"POST /api/v1/auth/password-reset/request -> {reset_request['status_code']}, артефакт: {reset_request['artifact_path']}",
            f"email_jobs reset token={state['reset_token']}, артефакт: {RAW_DIR / '03-auth-reset-email-job.json'}",
            f"POST /api/v1/auth/password-reset/confirm -> {reset_confirm['status_code']}, артефакт: {reset_confirm['artifact_path']}",
            f"POST /api/v1/auth/login old -> {login_old['status_code']}, артефакт: {login_old['artifact_path']}",
            f"POST /api/v1/auth/login new -> {login_new['status_code']}, артефакт: {login_new['artifact_path']}",
            f"POST /api/v1/auth/password-reset/confirm reuse -> {reset_reuse['status_code']}, артефакт: {reset_reuse['artifact_path']}",
        ],
    )

    catalog_page1 = api_request(
        "GET",
        "/api/v1/products?q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=1&limit=20",
        artifact_name="04-catalog-page1.json",
    )
    catalog_page2 = api_request(
        "GET",
        "/api/v1/products?q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=2&limit=20",
        artifact_name="04-catalog-page2.json",
    )
    catalog_items = catalog_page1["response_json"]["data"]["items"]
    previous_price = None
    all_filtered = True
    sorted_asc = True
    for item in catalog_items:
        haystack = f"{item.get('name', '')} {item.get('description', '')}".lower()
        if not (100 <= item["price"] <= 1000 and item["stock_qty"] > 0 and "headphones" in haystack):
            all_filtered = False
        if previous_price is not None and item["price"] < previous_price:
            sorted_asc = False
        previous_price = item["price"]
    catalog_ok = (
        catalog_page1["status_code"] == 200
        and catalog_page1["response_json"]["data"]["page"] == 1
        and catalog_page1["response_json"]["data"]["limit"] == 20
        and catalog_page1["response_json"]["data"]["total"] >= len(catalog_items)
        and catalog_page2["status_code"] == 200
        and all_filtered
        and sorted_asc
    )
    record_case(
        case_id="TC_API_CAT_001",
        priority="Высокий",
        name="Получение каталога товаров с поиском, ценовыми фильтрами, сортировкой и пагинацией",
        precondition="Каталог доступен через seeded данные и GET /api/v1/products.",
        steps=[
            "GET /api/v1/products ... page=1",
            "Проверка фильтров и сортировки",
            "GET /api/v1/products ... page=2",
        ],
        test_data={"query": "q=headphones&min_price=100&max_price=1000&in_stock=true&sort=price_asc&page=1&limit=20"},
        expected="200 OK; все элементы соответствуют фильтрам; price_asc соблюден; метаданные page/limit/total валидны.",
        actual=(
            f"page1 -> {catalog_page1['status_code']}, items={len(catalog_items)}, "
            f"page={catalog_page1['response_json']['data']['page']}, "
            f"limit={catalog_page1['response_json']['data']['limit']}, "
            f"total={catalog_page1['response_json']['data']['total']}, "
            f"filters_ok={all_filtered}, sorted_ok={sorted_asc}; "
            f"page2 -> {catalog_page2['status_code']}, items={len(catalog_page2['response_json']['data']['items'])}."
        ),
        status="Зачет" if catalog_ok else "Незачет",
        evidence=[
            f"GET /api/v1/products page=1 -> {catalog_page1['status_code']}, артефакт: {catalog_page1['artifact_path']}",
            f"GET /api/v1/products page=2 -> {catalog_page2['status_code']}, артефакт: {catalog_page2['artifact_path']}",
        ],
    )

    customer_headers = auth_header(state["customer_tokens"]["access_token"])
    fav_add = api_request("POST", f"/api/v1/favorites/{state['product']['id']}", headers=customer_headers, artifact_name="05-favorites-add.json")
    fav_list = api_request("GET", "/api/v1/favorites?page=1&limit=20", headers=customer_headers, artifact_name="05-favorites-list.json")
    fav_delete = api_request("DELETE", f"/api/v1/favorites/{state['product']['id']}", headers=customer_headers, artifact_name="05-favorites-delete.json")
    fav_list_after_delete = api_request("GET", "/api/v1/favorites?page=1&limit=20", headers=customer_headers, artifact_name="05-favorites-list-after-delete.json")
    fav_ids = [item.get("id") for item in response_items(fav_list)]
    fav_ids_after_delete = [item.get("id") for item in response_items(fav_list_after_delete)]
    fav_ok = (
        fav_add["status_code"] == 200
        and state["product"]["id"] in fav_ids
        and fav_delete["status_code"] == 200
        and state["product"]["id"] not in fav_ids_after_delete
    )
    record_case(
        case_id="TC_API_FAV_001",
        priority="Средний",
        name="Добавление товара в избранное и последующее удаление из списка favorites",
        precondition="Подтвержденный пользователь авторизован по bearer token.",
        steps=["POST /api/v1/favorites/{product_id}", "GET /api/v1/favorites", "DELETE /api/v1/favorites/{product_id}", "GET /api/v1/favorites"],
        test_data={"product_id": state["product"]["id"]},
        expected="POST 200; товар виден в favorites; DELETE 200; после удаления товар отсутствует в списке.",
        actual=f"add -> {fav_add['status_code']}; list after add -> {fav_list['status_code']}, present={state['product']['id'] in fav_ids}; delete -> {fav_delete['status_code']}; list after delete -> {fav_list_after_delete['status_code']}, present={state['product']['id'] in fav_ids_after_delete}.",
        status="Зачет" if fav_ok else "Незачет",
        evidence=[
            f"POST /api/v1/favorites/{state['product']['id']} -> {fav_add['status_code']}, артефакт: {fav_add['artifact_path']}",
            f"GET /api/v1/favorites -> {fav_list['status_code']}, артефакт: {fav_list['artifact_path']}",
            f"DELETE /api/v1/favorites/{state['product']['id']} -> {fav_delete['status_code']}, артефакт: {fav_delete['artifact_path']}",
            f"GET /api/v1/favorites after delete -> {fav_list_after_delete['status_code']}, артефакт: {fav_list_after_delete['artifact_path']}",
        ],
    )

    place_create = api_request("POST", "/api/v1/places", headers=customer_headers, body={"title": "Home", "address_text": "Moscow, Tverskaya 1", "lat": 55.7558, "lon": 37.6173}, artifact_name="06-places-create.json")
    state["place_crud_id"] = (response_data(place_create) or {}).get("id")
    place_update = api_request("PATCH", f"/api/v1/places/{state['place_crud_id']}", headers=customer_headers, body={"title": "QA Home", "address_text": "Moscow, Tverskaya 1, apt 2"}, artifact_name="06-places-update.json") if state["place_crud_id"] else {"status_code": 0, "artifact_path": "not-run"}
    place_list = api_request("GET", "/api/v1/places", headers=customer_headers, artifact_name="06-places-list.json")
    place_delete = api_request("DELETE", f"/api/v1/places/{state['place_crud_id']}", headers=customer_headers, artifact_name="06-places-delete.json") if state["place_crud_id"] else {"status_code": 0, "artifact_path": "not-run"}
    place_list_after_delete = api_request("GET", "/api/v1/places", headers=customer_headers, artifact_name="06-places-list-after-delete.json")
    places_before = response_data(place_list) if isinstance(response_data(place_list), list) else []
    places_after = response_data(place_list_after_delete) if isinstance(response_data(place_list_after_delete), list) else []
    updated_place = next((item for item in places_before if item.get("id") == state["place_crud_id"]), None)
    place_still_present = any(item.get("id") == state["place_crud_id"] for item in places_after)
    place_ok = (
        place_create["status_code"] == 201
        and place_update["status_code"] == 200
        and updated_place is not None
        and updated_place.get("title") == "QA Home"
        and place_delete["status_code"] == 200
        and not place_still_present
    )
    record_case(
        case_id="TC_API_PLACE_001",
        priority="Средний",
        name="Создание, редактирование и удаление адреса пользователя (places)",
        precondition="Подтвержденный пользователь авторизован.",
        steps=["POST /api/v1/places", "PATCH /api/v1/places/{id}", "GET /api/v1/places", "DELETE /api/v1/places/{id}", "GET /api/v1/places"],
        test_data={"place_id": state["place_crud_id"]},
        expected="Создание 201; update 200 и сохраняет новые поля; delete 200; после удаления адрес отсутствует в списке.",
        actual=f"create -> {place_create['status_code']}, id={state['place_crud_id']}; update -> {place_update['status_code']}; list after update -> title={updated_place['title'] if updated_place else None}; delete -> {place_delete['status_code']}; list after delete -> present={place_still_present}.",
        status="Зачет" if place_ok else "Незачет",
        evidence=[
            f"POST /api/v1/places -> {place_create['status_code']}, артефакт: {place_create['artifact_path']}",
            f"PATCH /api/v1/places/{state['place_crud_id']} -> {place_update['status_code']}, артефакт: {place_update['artifact_path']}",
            f"GET /api/v1/places -> {place_list['status_code']}, артефакт: {place_list['artifact_path']}",
            f"DELETE /api/v1/places/{state['place_crud_id']} -> {place_delete['status_code']}, артефакт: {place_delete['artifact_path']}",
            f"GET /api/v1/places after delete -> {place_list_after_delete['status_code']}, артефакт: {place_list_after_delete['artifact_path']}",
        ],
    )

    cart_clear = api_request("DELETE", "/api/v1/cart", headers=customer_headers, artifact_name="07-cart-clear.json")
    cart_add = api_request("POST", "/api/v1/cart/items", headers=customer_headers, body={"product_id": state["product"]["id"], "quantity": 2}, artifact_name="07-cart-add.json")
    cart_get_1 = api_request("GET", "/api/v1/cart", headers=customer_headers, artifact_name="07-cart-get-1.json")
    cart_patch = api_request("PATCH", f"/api/v1/cart/items/{state['product']['id']}", headers=customer_headers, body={"quantity": 4}, artifact_name="07-cart-patch.json")
    cart_get_2 = api_request("GET", "/api/v1/cart", headers=customer_headers, artifact_name="07-cart-get-2.json")
    cart_delete = api_request("DELETE", f"/api/v1/cart/items/{state['product']['id']}", headers=customer_headers, artifact_name="07-cart-delete-item.json")
    cart_get_3 = api_request("GET", "/api/v1/cart", headers=customer_headers, artifact_name="07-cart-get-3.json")
    cart_items_1 = response_items(cart_get_1)
    cart_items_2 = response_items(cart_get_2)
    cart_items_3 = response_items(cart_get_3)
    cart_item_1 = next((item for item in cart_items_1 if item.get("product_id") == state["product"]["id"]), None)
    cart_item_2 = next((item for item in cart_items_2 if item.get("product_id") == state["product"]["id"]), None)
    cart_data_1 = response_data(cart_get_1) or {}
    cart_data_2 = response_data(cart_get_2) or {}
    cart_ok = (
        cart_clear["status_code"] == 200
        and cart_add["status_code"] == 201
        and cart_item_1 is not None
        and cart_item_1.get("quantity") == 2
        and cart_patch["status_code"] == 200
        and cart_item_2 is not None
        and cart_item_2.get("quantity") == 4
        and cart_delete["status_code"] == 200
        and len(cart_items_3) == 0
    )
    record_case(
        case_id="TC_API_CART_001",
        priority="Высокий",
        name="Добавление товара в корзину, изменение количества и удаление позиции",
        precondition=f"Есть активный товар {state['product']['id']} с stock_qty={state['product']['stock_qty']}.",
        steps=["DELETE /api/v1/cart", "POST /api/v1/cart/items", "GET /api/v1/cart", "PATCH /api/v1/cart/items/{product_id}", "GET /api/v1/cart", "DELETE /api/v1/cart/items/{product_id}", "GET /api/v1/cart"],
        test_data={"product_id": state["product"]["id"], "add_quantity": 2, "update_quantity": 4},
        expected="Очистка 200; добавление 201; PATCH меняет quantity и суммы; после удаления корзина пустая.",
        actual=f"clear -> {cart_clear['status_code']}; add -> {cart_add['status_code']}; get1 quantity={cart_item_1['quantity'] if cart_item_1 else None}, total_items={cart_data_1.get('total_items')}, total_amount={cart_data_1.get('total_amount')}; patch -> {cart_patch['status_code']}; get2 quantity={cart_item_2['quantity'] if cart_item_2 else None}, total_items={cart_data_2.get('total_items')}, total_amount={cart_data_2.get('total_amount')}; delete -> {cart_delete['status_code']}; get3 items={len(cart_items_3)}.",
        status="Зачет" if cart_ok else "Незачет",
        evidence=[
            f"DELETE /api/v1/cart -> {cart_clear['status_code']}, артефакт: {cart_clear['artifact_path']}",
            f"POST /api/v1/cart/items -> {cart_add['status_code']}, артефакт: {cart_add['artifact_path']}",
            f"GET /api/v1/cart after add -> {cart_get_1['status_code']}, артефакт: {cart_get_1['artifact_path']}",
            f"PATCH /api/v1/cart/items/{state['product']['id']} -> {cart_patch['status_code']}, артефакт: {cart_patch['artifact_path']}",
            f"GET /api/v1/cart after patch -> {cart_get_2['status_code']}, артефакт: {cart_get_2['artifact_path']}",
            f"DELETE /api/v1/cart/items/{state['product']['id']} -> {cart_delete['status_code']}, артефакт: {cart_delete['artifact_path']}",
            f"GET /api/v1/cart after delete -> {cart_get_3['status_code']}, артефакт: {cart_get_3['artifact_path']}",
        ],
    )

    order_place_create = api_request("POST", "/api/v1/places", headers=customer_headers, body={"title": "Order Place", "address_text": "Moscow, Arbat 10", "lat": 55.7522, "lon": 37.5924}, artifact_name="08-order-place-create.json")
    state["order_place_id"] = (response_data(order_place_create) or {}).get("id")
    cart_add_order = api_request("POST", "/api/v1/cart/items", headers=customer_headers, body={"product_id": state["product"]["id"], "quantity": 2}, artifact_name="08-order-cart-add.json")
    order_create = api_request("POST", "/api/v1/orders", headers=customer_headers, body={"place_id": state["order_place_id"]}, artifact_name="08-order-create.json") if state["order_place_id"] else {"status_code": 0, "artifact_path": "not-run"}
    state["order_id"] = (response_data(order_create) or {}).get("id")
    orders_list = api_request("GET", "/api/v1/orders?page=1&limit=20", headers=customer_headers, artifact_name="08-order-list.json")
    order_get = api_request("GET", f"/api/v1/orders/{state['order_id']}", headers=customer_headers, artifact_name="08-order-get.json") if state["order_id"] else {"status_code": 0, "artifact_path": "not-run"}
    order_ids = [item.get("id") for item in response_items(orders_list)]
    order_get_data = response_data(order_get) or {}
    order_ok = (
        order_place_create["status_code"] == 201
        and cart_add_order["status_code"] == 201
        and order_create["status_code"] == 201
        and state["order_id"] in order_ids
        and order_get["status_code"] == 200
        and order_get_data.get("place_id") == state["order_place_id"]
    )
    record_case(
        case_id="TC_API_ORDER_001",
        priority="Высокий",
        name="Оформление заказа из непустой корзины и проверка появления заказа в истории",
        precondition="Подготовлен отдельный place_id и корзина с товаром.",
        steps=["POST /api/v1/places", "POST /api/v1/cart/items", "POST /api/v1/orders", "GET /api/v1/orders", "GET /api/v1/orders/{id}"],
        test_data={"place_id": state["order_place_id"], "product_id": state["product"]["id"], "quantity": 2},
        expected="Checkout 201; ответ содержит order_id; заказ виден в истории; GET /orders/{id} возвращает корректные детали.",
        actual=f"place create -> {order_place_create['status_code']}, place_id={state['order_place_id']}; cart add -> {cart_add_order['status_code']}; order create -> {order_create['status_code']}, order_id={state['order_id']}; orders list -> {orders_list['status_code']}, in_list={state['order_id'] in order_ids}; order get -> {order_get['status_code']}, items={len(order_get_data.get('items', []))}, total_amount={order_get_data.get('total_amount')}.",
        status="Зачет" if order_ok else "Незачет",
        evidence=[
            f"POST /api/v1/places -> {order_place_create['status_code']}, артефакт: {order_place_create['artifact_path']}",
            f"POST /api/v1/cart/items -> {cart_add_order['status_code']}, артефакт: {cart_add_order['artifact_path']}",
            f"POST /api/v1/orders -> {order_create['status_code']}, артефакт: {order_create['artifact_path']}",
            f"GET /api/v1/orders -> {orders_list['status_code']}, артефакт: {orders_list['artifact_path']}",
            f"GET /api/v1/orders/{state['order_id']} -> {order_get['status_code']}, артефакт: {order_get['artifact_path']}",
        ],
    )

    login_second_session = login_user(state["customer_email"], state["customer_new_password"], "09-auth-login-second-session.json")
    state["customer_tokens_2"] = (response_data(login_second_session) or {}).get("tokens")
    sessions_before = api_request("GET", "/api/v1/auth/sessions", headers=customer_headers, artifact_name="09-sessions-before.json")
    sessions = response_data(sessions_before) if isinstance(response_data(sessions_before), list) else []
    secondary = next((item for item in sessions if not item.get("is_current")), sessions[-1] if sessions else None)
    state["secondary_session_id"] = secondary["id"] if secondary else None
    session_delete = api_request("DELETE", f"/api/v1/auth/sessions/{state['secondary_session_id']}", headers=customer_headers, artifact_name="09-sessions-delete.json") if state["secondary_session_id"] else {"status_code": 0, "artifact_path": "not-run"}
    sessions_after = api_request("GET", "/api/v1/auth/sessions", headers=customer_headers, artifact_name="09-sessions-after.json")
    refresh_revoked = api_request("POST", "/api/v1/auth/refresh", body={"refresh_token": state["customer_tokens_2"]["refresh_token"]}, artifact_name="09-sessions-refresh-revoked.json") if state["customer_tokens_2"] else {"status_code": 0, "artifact_path": "not-run"}
    session_ids_after = [item.get("id") for item in (response_data(sessions_after) if isinstance(response_data(sessions_after), list) else [])]
    session_ok = (
        login_second_session["status_code"] == 200
        and len(sessions) >= 2
        and session_delete["status_code"] == 200
        and state["secondary_session_id"] not in session_ids_after
        and refresh_revoked["status_code"] == 401
    )
    record_case(
        case_id="TC_API_SESS_001",
        priority="Высокий",
        name="Просмотр активных сессий и отзыв конкретной refresh-сессии пользователя",
        precondition="Созданы две bearer-сессии одного пользователя.",
        steps=["POST /api/v1/auth/login для второй сессии", "GET /api/v1/auth/sessions", "DELETE /api/v1/auth/sessions/{id}", "GET /api/v1/auth/sessions", "POST /api/v1/auth/refresh с revoked refresh_token"],
        test_data={"session_count_before": len(sessions), "revoked_session_id": state["secondary_session_id"]},
        expected="Список сессий 200 и минимум две записи; удаление 200; revoked session отсутствует в списке; refresh этой сессией дает ошибку авторизации.",
        actual=f"second login -> {login_second_session['status_code']}; sessions before -> {sessions_before['status_code']}, count={len(sessions)}; delete -> {session_delete['status_code']}, revoked_session_id={state['secondary_session_id']}; sessions after -> {sessions_after['status_code']}, present={state['secondary_session_id'] in session_ids_after}; refresh revoked -> {refresh_revoked['status_code']}.",
        status="Зачет" if session_ok else "Незачет",
        evidence=[
            f"POST /api/v1/auth/login second session -> {login_second_session['status_code']}, артефакт: {login_second_session['artifact_path']}",
            f"GET /api/v1/auth/sessions -> {sessions_before['status_code']}, артефакт: {sessions_before['artifact_path']}",
            f"DELETE /api/v1/auth/sessions/{state['secondary_session_id']} -> {session_delete['status_code']}, артефакт: {session_delete['artifact_path']}",
            f"GET /api/v1/auth/sessions after delete -> {sessions_after['status_code']}, артефакт: {sessions_after['artifact_path']}",
            f"POST /api/v1/auth/refresh revoked -> {refresh_revoked['status_code']}, артефакт: {refresh_revoked['artifact_path']}",
        ],
    )

    seller_login = login_user("merchant-urbanwave@seed.marketplace.local", "Seller123", "10-seller-login.json")
    state["seller_tokens"] = (response_data(seller_login) or {}).get("tokens")
    seller_headers = auth_header(state["seller_tokens"]["access_token"]) if state["seller_tokens"] else {}
    seller_dashboard = api_request("GET", "/api/v1/seller/dashboard", headers=seller_headers, artifact_name="10-seller-dashboard.json")
    seller_payload = {
        "category_id": state["seller_category_id"],
        "name": f"QA Seller Product {state['run_id']}",
        "slug": f"qa-seller-product-{state['run_id']}",
        "description": "Temporary QA seller product",
        "price": 1234,
        "currency": "RUB",
        "sku": f"QA-SKU-{state['run_id']}",
        "image_url": f"https://example.com/qa-seller-product-{state['run_id']}.jpg",
        "images": [f"https://example.com/qa-seller-product-{state['run_id']}.jpg"],
        "brand": "UrbanWave",
        "unit": "piece",
        "specs": {"qa": "true", "run_id": state["run_id"]},
        "stock_qty": 7,
        "is_active": True,
    }
    seller_create = api_request("POST", "/api/v1/seller/products", headers=seller_headers, body=seller_payload, artifact_name="10-seller-create-product.json")
    state["seller_product_id"] = (response_data(seller_create) or {}).get("id")
    seller_patch = api_request("PATCH", f"/api/v1/seller/products/{state['seller_product_id']}/stock", headers=seller_headers, body={"stock_qty": 11}, artifact_name="10-seller-patch-stock.json") if state["seller_product_id"] else {"status_code": 0, "artifact_path": "not-run", "response_json": {}}
    seller_list = api_request("GET", "/api/v1/seller/products?page=1&limit=20&q=QA%20Seller%20Product", headers=seller_headers, artifact_name="10-seller-list.json")
    seller_delete = api_request("DELETE", f"/api/v1/seller/products/{state['seller_product_id']}", headers=seller_headers, artifact_name="10-seller-delete-product.json") if state["seller_product_id"] else {"status_code": 0, "artifact_path": "not-run"}
    listed_product = next((item for item in response_items(seller_list) if item.get("id") == state["seller_product_id"]), None)
    seller_patch_data = response_data(seller_patch) or {}
    seller_ok = (
        seller_login["status_code"] == 200
        and seller_dashboard["status_code"] == 200
        and seller_create["status_code"] == 201
        and seller_patch["status_code"] == 200
        and listed_product is not None
        and listed_product.get("stock_qty") == 11
        and seller_delete["status_code"] == 200
    )
    record_case(
        case_id="TC_API_SELL_001",
        priority="Средний",
        name="Доступ продавца к seller dashboard и базовое управление собственным товаром",
        precondition="Использован seeded seller user с активным seller profile.",
        steps=["POST /api/v1/auth/login", "GET /api/v1/seller/dashboard", "POST /api/v1/seller/products", "PATCH /api/v1/seller/products/{id}/stock", "GET /api/v1/seller/products", "DELETE /api/v1/seller/products/{id}"],
        test_data={"seller_email": "merchant-urbanwave@seed.marketplace.local", "category_id": state["seller_category_id"], "seller_product_id": state["seller_product_id"]},
        expected="Dashboard 200; создание товара 201; update stock 200; товар виден в списке с новым stock_qty; архивация 200.",
        actual=f"seller login -> {seller_login['status_code']}; dashboard -> {seller_dashboard['status_code']}; create -> {seller_create['status_code']}, product_id={state['seller_product_id']}; stock patch -> {seller_patch['status_code']}, stock_qty={seller_patch_data.get('stock_qty')}; seller list -> {seller_list['status_code']}, listed_stock={listed_product['stock_qty'] if listed_product else None}; delete -> {seller_delete['status_code']}.",
        status="Зачет" if seller_ok else "Незачет",
        evidence=[
            f"POST /api/v1/auth/login seller -> {seller_login['status_code']}, артефакт: {seller_login['artifact_path']}",
            f"GET /api/v1/seller/dashboard -> {seller_dashboard['status_code']}, артефакт: {seller_dashboard['artifact_path']}",
            f"POST /api/v1/seller/products -> {seller_create['status_code']}, артефакт: {seller_create['artifact_path']}",
            f"PATCH /api/v1/seller/products/{state['seller_product_id']}/stock -> {seller_patch['status_code']}, артефакт: {seller_patch['artifact_path']}",
            f"GET /api/v1/seller/products -> {seller_list['status_code']}, артефакт: {seller_list['artifact_path']}",
            f"DELETE /api/v1/seller/products/{state['seller_product_id']} -> {seller_delete['status_code']}, артефакт: {seller_delete['artifact_path']}",
        ],
    )

    api_request("DELETE", f"/api/v1/favorites/{state['product']['id']}", headers=customer_headers, artifact_name="99-cleanup-favorites.json")
    api_request("DELETE", "/api/v1/cart", headers=customer_headers, artifact_name="99-cleanup-cart.json")
    if state["secondary_session_id"]:
        api_request("DELETE", f"/api/v1/auth/sessions/{state['secondary_session_id']}", headers=customer_headers, artifact_name="99-cleanup-session.json")
    if state["seller_product_id"]:
        api_request("DELETE", f"/api/v1/seller/products/{state['seller_product_id']}", headers=seller_headers, artifact_name="99-cleanup-seller-product.json")

    json_path, md_path = write_reports()
    print(json.dumps({"base_url": state["base_url"], "results_json": json_path, "report_md": md_path}, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        fatal_path = save_json_artifact(
            "fatal-error.json",
            {
                "message": str(exc),
                "type": type(exc).__name__,
            },
        )
        print(json.dumps({"error": str(exc), "fatal_artifact": fatal_path}, ensure_ascii=False, indent=2), file=sys.stderr)
        raise
