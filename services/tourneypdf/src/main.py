import asyncio
import json
import os
import signal
import tempfile
import zipfile

import nats

from generator import ScorecardCreator, URLNotUniqueException
from fetch_tourney import get_tournament


async def message_handler(msg):
    subject = msg.subject
    reply = msg.reply
    data = msg.data.decode()
    print(
        "Received a message on '{subject} {reply}': {data}".format(
            subject=subject, reply=reply, data=data
        )
    )
    try:
        payload = json.loads(data)
    except Exception as e:
        print("Failed to load JSON data", e)
        return

    try:
        t = get_tournament(payload.get("slug", ""))
    except Exception as e:
        print("Failed to fetch tourney", e)
        return

    creator = ScorecardCreator(
        t,
        payload.get("show_opponents", False),
        payload.get("show_seeds", False),
        payload.get("show_qrcode", False),
    )

    with tempfile.TemporaryDirectory() as temp_dir:
        print("Using temp_dir to save files:", temp_dir)

        creator.set_output_path(temp_dir)
        try:
            creator.gen_scorecards()
        except Exception as e:
            print("Failed to generate scorecards", e)
            return

        # Zipping the PDF files in the temporary directory
        zip_path = os.path.join(temp_dir, "output.zip")
        with zipfile.ZipFile(zip_path, "w") as zipf:
            for file in os.listdir(temp_dir):
                if file.endswith(".pdf"):
                    zipf.write(os.path.join(temp_dir, file), file)

        # Read the zip file as bytes
        with open(zip_path, "rb") as f:
            zip_bytes = f.read()

        # Send the zip file back via NATS reply channel
        if reply:
            print("Sending zip back via", reply)
            await msg.respond(zip_bytes)

        print("Succeeded; created pdfs")


async def main():
    nc = await nats.connect(os.environ["NATS_URL"])

    sub = await nc.subscribe("tourneypdf", cb=message_handler)
    print("Subscribed to tourneypdf channel")
    # Use an asyncio.Event to manage the loop termination
    loop_done = asyncio.Event()

    # Define a function to stop the loop on signal
    def signal_handler():
        loop_done.set()

    # Register the signal handler for SIGINT
    loop = asyncio.get_running_loop()
    for signame in {"SIGINT", "SIGTERM"}:
        loop.add_signal_handler(getattr(signal, signame), signal_handler)

    # Wait until the event is set
    await loop_done.wait()
    print("loop_done")
    # Cleanup: Unsubscribe, drain connection
    await sub.unsubscribe()
    await nc.drain()
    print("Exiting main()")


if __name__ == "__main__":
    asyncio.run(main())
