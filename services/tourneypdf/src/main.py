import base64
import json
import os
import sys
import tempfile
import zipfile
import traceback

os.makedirs("/tmp/fontconfig", exist_ok=True)
os.environ["FONTCONFIG_CACHE"] = "/tmp/fontconfig"

from generator import ScorecardCreator
from fetch_tourney import get_tournament


def message_handler(msg):
    try:
        t = get_tournament(msg.get("id", ""))
    except Exception as e:
        print("Failed to fetch tourney", e)
        raise

    creator = ScorecardCreator(
        t,
        msg.get("showOpponents", False),
        msg.get("showSeeds", False),
        msg.get("showQrCode", False),
    )
    print("Got tournament with id:", msg.get("id"))
    print("Tournament slug:", t["meta"]["metadata"]["slug"])
    with tempfile.TemporaryDirectory() as temp_dir:
        print("Using temp_dir to save files:", temp_dir)

        creator.set_output_path(temp_dir)
        try:
            creator.gen_scorecards()
        except Exception as e:
            print("Failed to generate scorecards", e)
            traceback.print_exc()
            raise

        # Zipping the PDF files in the temporary directory
        zip_path = os.path.join(temp_dir, "output.zip")
        with zipfile.ZipFile(zip_path, "w") as zipf:
            for file in os.listdir(temp_dir):
                if file.endswith(".pdf"):
                    zipf.write(os.path.join(temp_dir, file), file)

        # Read the zip file as bytes
        with open(zip_path, "rb") as f:
            zip_bytes = f.read()

        return zip_bytes


def lambda_handler(event, context):
    try:
        zip_bytes = message_handler(event)
    except Exception as e:
        return {"statusCode": 500, "body": json.dumps({"error": str(e)})}

    return {"statusCode": 200, "payload": base64.b64encode(zip_bytes)}
