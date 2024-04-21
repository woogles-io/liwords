# Only works with woggles tournaments
import os
import requests
import json


class FetchTourneyError(Exception):
    pass


def get_tournament(slug: str):
    endpoint = os.environ["API_ENDPOINT"]
    tm = requests.post(
        f"{endpoint}/tournament_service.TournamentService/GetTournamentMetadata",
        headers={"content-type": "application/json"},
        data=json.dumps({"slug": slug}),
    )
    if tm.status_code != 200:
        raise FetchTourneyError("Error fetching metadata: " + str(tm.status_code))

    tid = tm.json()["metadata"]["id"]
    t = requests.post(
        f"{endpoint}/tournament_service.TournamentService/GetTournament",
        headers={"content-type": "application/json"},
        data=json.dumps({"id": tid}),
    )

    return {"t": t.json(), "meta": tm.json()}


def player_by_idx(tourney, div, idx):
    p = tourney["divisions"][div]["players"]["persons"][idx]
    return p["id"].split(":")[1], p.get("rating", 0)
