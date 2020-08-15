"""
email passwords to users.

"""
import csv
import os
import sys

import requests


MAILGUN_KEY = os.getenv("MAILGUN_KEY")

with open("email_template.txt") as f:
    email_template = f.read()


def send_email(recipient, username, password):
    return requests.post(
        "https://api.mailgun.net/v3/mg.woogles.io/messages",
        auth=("api", MAILGUN_KEY),
        data={
            "from": "Woogles <mailgun@mg.woogles.io>",
            "to": [recipient],
            "subject": "Welcome to the Woogles.io alpha!",
            "text": email_template.format(
                username=username, password=password
            ),
        },
    )


def emailer(incsv):
    with open(incsv, newline="") as f:
        reader = csv.reader(f)
        for row in reader:
            if not row:
                continue
            if row[0] == "username" and row[1] == "email":
                continue

            username, email, password = row
            resp = send_email(email, username, password)
            if resp.status_code != 200:
                print(
                    "Mailgun request failed",
                    email,
                    resp.status_code,
                    resp.text,
                )


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python send_emails.py <input.csv>")
        sys.exit(1)
    emailer(sys.argv[1])
