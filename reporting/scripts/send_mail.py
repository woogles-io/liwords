#!/usr/bin/env python3
"""Send a plain-text email (optionally with attachments) via Gmail SMTP.

Credentials (GMAIL_USER, GMAIL_APP_PASSWORD) are read from reporting/.env,
alongside the DB connection settings. Used by monthly_report.sh to deliver
the monthly Woogles reporting email and failure alerts.
"""
import argparse
import mimetypes
import os
import smtplib
import sys
from email.message import EmailMessage
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ENV_FILE = SCRIPT_DIR.parent / ".env"


def load_env(path):
    env = {}
    if not path.is_file():
        return env
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        env[key.strip()] = value.strip()
    return env


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--subject", required=True)
    body_group = parser.add_mutually_exclusive_group(required=True)
    body_group.add_argument("--body")
    body_group.add_argument("--body-file", help="Read the body from this file")
    parser.add_argument("--attach", action="append", default=[],
                        help="File to attach (repeatable)")
    parser.add_argument("--html-file", default=None,
                        help="HTML alternative body; plain body remains the fallback")
    parser.add_argument("--inline", action="append", default=[],
                        help="Image to embed in the HTML body (repeatable); "
                             "referenced as cid:<filename-without-extension>")
    parser.add_argument("--to", default=None, help="Defaults to GMAIL_USER")
    args = parser.parse_args()

    env = {**load_env(ENV_FILE), **os.environ}
    gmail_user = env.get("GMAIL_USER")
    gmail_app_password = env.get("GMAIL_APP_PASSWORD")
    if not gmail_user or not gmail_app_password:
        print("GMAIL_USER / GMAIL_APP_PASSWORD not set in reporting/.env", file=sys.stderr)
        sys.exit(1)

    to_addr = args.to or gmail_user
    body = args.body if args.body is not None else Path(args.body_file).read_text()

    msg = EmailMessage()
    msg["Subject"] = args.subject
    msg["From"] = gmail_user
    msg["To"] = to_addr
    msg.set_content(body)

    if args.html_file:
        msg.add_alternative(Path(args.html_file).read_text(), subtype="html")
        html_part = msg.get_body(preferencelist=("html",))
        for inline_path in args.inline:
            path = Path(inline_path)
            if not path.is_file():
                print(f"Inline image not found: {path}", file=sys.stderr)
                sys.exit(1)
            html_part.add_related(path.read_bytes(), maintype="image",
                                  subtype=path.suffix.lstrip(".") or "png",
                                  cid=f"<{path.stem}>")

    for attach_path in args.attach:
        path = Path(attach_path)
        if not path.is_file():
            print(f"Attachment not found: {path}", file=sys.stderr)
            sys.exit(1)
        ctype, _ = mimetypes.guess_type(path.name)
        maintype, subtype = (ctype or "application/octet-stream").split("/", 1)
        msg.add_attachment(path.read_bytes(), maintype=maintype,
                           subtype=subtype, filename=path.name)

    with smtplib.SMTP("smtp.gmail.com", 587) as smtp:
        smtp.starttls()
        smtp.login(gmail_user, gmail_app_password)
        smtp.send_message(msg)


if __name__ == "__main__":
    main()
