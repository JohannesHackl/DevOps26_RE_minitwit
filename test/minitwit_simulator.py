"""
Call me for example like:

$ python minitwit_simulator.py "http://localhost:5001"
"""

import traceback
import os
import csv
import sys
import json
import http
import socket
import base64
import requests
from time import sleep
from datetime import datetime
from contextlib import closing
import sqlite3
from collections import defaultdict


CSV_FILENAME = "./test/minitwit_scenario.csv"
USERNAME = "simulator"
PWD = "super_safe!"
CREDENTIALS = ":".join([USERNAME, PWD]).encode("ascii")
ENCODED_CREDENTIALS = base64.b64encode(CREDENTIALS).decode()
HEADERS = {
    "Connection": "close",
    "Content-Type": "application/json",
    f"Authorization": f"Basic {ENCODED_CREDENTIALS}",
}


def log_failure(failures, action, reason, status_code=None):
    ts_str = datetime.strftime(datetime.utcnow(), "%Y-%m-%d %H:%M:%S")
    entry = {
        "time": ts_str,
        "command_id": action["latest"],
        "command": action["post_type"],
        "status": status_code,
        "reason": reason,
    }
    failures.append(entry)
    print(",".join([ts_str, str(action["latest"]), action["post_type"], reason, str(status_code)]))

def get_actions():

    # read scenario .csv and parse to a list of lists
    with open(CSV_FILENAME, "r", encoding="utf-8") as f:
        reader = csv.reader(f, delimiter="\t", quotechar=None)

        # for each line in .csv
        for line in reader:
            try:
                # we know that the command string is always the fourth element
                command = line[3]

                command_id = int(line[0])
                delay = int(line[1])
                user = line[4]
                if command == "register":
                    email = line[5]
                    user_pwd = line[-1]

                    item = {
                        "latest": command_id,
                        "post_type": command,
                        "username": user,
                        "email": email,
                        "pwd": user_pwd,
                    }

                    yield item, delay

                elif command == "tweet":
                    tweet_content = line[5]
                    item = {
                        "latest": command_id,
                        "post_type": command,
                        "username": user,
                        "content": tweet_content,
                    }

                    yield item, delay

                elif command == "follow":
                    user_to_follow = line[5]
                    item = {
                        "latest": command_id,
                        "post_type": command,
                        "username": user,
                        "follow": user_to_follow,
                    }
                    yield item, delay

                elif command == "unfollow":
                    user_to_unfollow = line[5]

                    item = {
                        "latest": command_id,
                        "post_type": command,
                        "username": user,
                        "unfollow": user_to_unfollow,
                    }
                    yield item, delay

                else:
                    # This should never happen and can likely be removed to
                    # make parsing for plot generation later easier
                    print("Unknown type found: (" + command + ")")

            except Exception as e:
                print("========================================")
                print(traceback.format_exc())


def main(host):
    failures = []
    

    totals = defaultdict(int)
    for action, delay in get_actions():
        command = action["post_type"]
        if command in {"register", "tweet", "follow", "unfollow"}:
            totals[command] += 1
        else:
            totals[command + "--UNEXPECTED"] += 1
        try:
            # SWITCH ON TYPE
            command = action["post_type"]
            response = None

            if command == "register":

                # build url for request
                url = f"{host}/register"

                # Set parameters: latest
                params = {"latest": action["latest"]}
                # Set data: username, email, pwd
                data = {
                    "username": action["username"],
                    "email": action["email"],
                    "pwd": action["pwd"],
                }

                response = requests.post(
                    url,
                    data=json.dumps(data),
                    params=params,
                    headers=HEADERS,
                    timeout=0.3,
                )

                # error handling (204 success, 400 user exists)
                # 400 user exists already but not an error to log
                if not (
                    (response.status_code == 204)
                    or (response.status_code == 400)
                ):
                    log_failure(failures, action, "Unexpected", response.status_code)

                response.close()

            elif command == "msgs":

                # LIST method. Not used atm.
                # build url for request
                url = f"{host}/msgs"

                # Set parameters: latest & no (amount)
                params = {"latest": action["latest"], "no": action["no"]}

                response = requests.post(
                    url, params=params, headers=HEADERS, timeout=0.3
                )

                # error handling (200 success, 403 failure (no headers))

                # 403 bad request
                if response.status_code != 200:
                    log_failure(failures, action, "Bad request", response.status_code)

                response.close()

            elif command == "follow":

                # build url for request
                username = action["username"]
                url = f"{host}/fllws/{username}"

                # Set parameters: latest
                params = {"latest": action["latest"]}
                # Set data: content
                data = {"follow": action["follow"]}  # value for user to follow

                response = requests.post(
                    url,
                    data=json.dumps(data),
                    params=params,
                    headers=HEADERS,
                    timeout=0.3,
                )

                # error handling (204 success, 403 failure, 404 Not Found no user id)

                # 403 unauthorized or 404 Not Found
                if response.status_code != 204:
                    reason = "Unauthorized" if response.status_code == 403 else "Not found: User id"
                    log_failure(failures, action, reason, response.status_code)

                response.close()

            elif command == "unfollow":

                # build url for request
                username = action["username"]
                url = f"{host}/fllws/{username}"
                # Set parameters: latest
                params = {"latest": action["latest"]}
                # Set data: content
                # value for user to follow
                data = {"unfollow": action["unfollow"]}

                response = requests.post(
                    url,
                    data=json.dumps(data),
                    params=params,
                    headers=HEADERS,
                    timeout=0.3,
                )

                # error handling (204 success, 403 failure, 404 Not Found no user id)

                # 403 unauthorized or 404 Not Found
                if response.status_code != 204:
                    reason = "Unauthorized" if response.status_code == 403 else "Not found: User id"
                    log_failure(failures, action, reason, response.status_code)

                response.close()

            elif command == "tweet":

                # build url for request
                username = action["username"]
                url = f"{host}/msgs/{username}"
                # Set parameters: latest
                params = {"latest": action["latest"]}
                # Set data: content
                data = {"content": action["content"]}

                response = requests.post(
                    url,
                    data=json.dumps(data),
                    params=params,
                    headers=HEADERS,
                    timeout=0.3,
                )

                # error handling (204 success, 403 failure)
                # 403 unauthorized
                if response.status_code != 204:
                    log_failure(failures, action, "Unauthorized", response.status_code)

                response.close()

            else:
                log_failure(failures, action, "FATAL: Unknown message type")
                

        except requests.exceptions.ConnectionError as e:
            log_failure(failures, action, "ConnectionError")
        except requests.exceptions.ReadTimeout as e:
            log_failure(failures, action, "ReadTimeout")
        except Exception as e:
            print("========================================")
            print(traceback.format_exc())
            log_failure(failures, action, type(e).__name__)

        sleep(delay / (1000 * 100000))

    return failures, totals


if __name__ == "__main__":
    host = sys.argv[1]

    failures, totals = main(host)

    total = sum(totals.values())
    if total > 0 and len(failures) / total > 0.05: # 5% errors max to account for false error due to timeouts.
        from collections import Counter
        failure_counts = Counter(f["command"] for f in failures)
        print(f"\n=== {len(failures)} failure(s) ===")
        for command, total in totals.items():
            failed = failure_counts.get(command, 0)
            print(f"  {command}: {failed}/{total} failed")
        sys.exit(1)
