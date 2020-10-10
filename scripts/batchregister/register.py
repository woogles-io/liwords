"""
A script to batch register a list of users.
It will provide them with passwords.

"""
import csv
import os
import secrets
import sys

import requests


registration_api = (
    "https://woogles.io/twirp/user_service.RegistrationService/Register"
)
registration_code = os.getenv("REGISTRATION_CODE")


alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"


def gen_pw(length=10):
    return "".join(secrets.choice(alphabet) for i in range(length))


def register(incsv, outcsv):
    """
    Pass in a csv file with username,email

    Returns a csv file with username,email,temp_password

    """
    with open(outcsv, "w", newline="") as fout:
        csvwriter = csv.writer(fout)
        csvwriter.writerow(["username", "email", "password"])
        with open(incsv, newline="") as f:
            reader = csv.DictReader(f)
            for row in reader:
                if not row:
                    continue
                password = gen_pw()
                # use twirp api to register
                if row["username"] == "" or row["email"] == "":
                    print("missing info:", row["username"], row["email"])
                    continue
                resp = requests.post(
                    registration_api,
                    headers={"Content-Type": "application/json"},
                    json={
                        "username": row["username"],
                        "email": row["email"],
                        "password": password,
                        "registration_code": registration_code,
                    },
                )
                if resp.status_code != 200:
                    print(resp.status_code, resp.text)
                    print(
                        "failed to register: ", row["username"], row["email"]
                    )
                    continue

                csvwriter.writerow([row["username"], row["email"], password])


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python register.py <filename.csv> <output.csv>")
        sys.exit(1)

    register(sys.argv[1], sys.argv[2])
