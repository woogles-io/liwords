import sys
import tempfile

from generator import ScorecardCreator
from fetch_tourney import get_tournament

if __name__ == "__main__":
    # invoke like
    # API_ENDPOINT=http://localhost/api python local_tester.py <tid>

    t = get_tournament(sys.argv[1])
    creator = ScorecardCreator(t, True, True, True)
    with tempfile.TemporaryDirectory(delete=False) as temp_dir:
        print("Using temp_dir to save files:", temp_dir)

        creator.set_output_path(temp_dir)
        try:
            creator.gen_scorecards()
        except Exception as e:
            print("Failed to generate scorecards", e)
            raise

    print("Successfully generated scorecards")
