import json
import sys

# Check for the required positional argument
if len(sys.argv) != 2:
    print("Usage: python generate_go_function.py <path_to_request.json>")
    sys.exit(1)

# Load the JSON data from the provided file path
json_file = sys.argv[1]
try:
    with open(json_file, "r") as f:
        request_data = json.load(f)
except Exception as e:
    print(f"Error reading JSON file: {e}")
    sys.exit(1)

# Extract relevant data
player_names = request_data["playerNames"]
player_classes = request_data["playerClasses"]
division_pairings = request_data["divisionPairings"]
division_results = request_data["divisionResults"]
class_prizes = request_data["classPrizes"]
gibson_spread = request_data["gibsonSpread"]
control_loss_threshold = request_data["controlLossThreshold"]
hopefulness_threshold = request_data["hopefulnessThreshold"]
all_players = request_data["allPlayers"]
valid_players = request_data["validPlayers"]
rounds = request_data["rounds"]
place_prizes = request_data["placePrizes"]
division_sims = request_data["divisionSims"]
control_loss_sims = request_data["controlLossSims"]
use_control_loss = request_data["useControlLoss"]
allow_repeat_byes = request_data["allowRepeatByes"]
removed_players = request_data.get("removedPlayers", [])
seed = request_data["seed"]

# Function name based on JSON content
function_name = "CreateCustomPairRequest"

# Generate the Go function
go_function = f"""
func {function_name}() *pb.PairRequest {{
    request := &pb.PairRequest{{
        PairMethod:           pb.PairMethod_COP,
        PlayerNames:          []string{{{", ".join(f'"{name}"' for name in player_names)}}},
        PlayerClasses:        []int32{{{", ".join(map(str, player_classes))}}},
        DivisionPairings: []*pb.RoundPairings{{
            {", ".join(f"{{Pairings: []int32{{{', '.join(map(str, round['pairings']))}}}}}" for round in division_pairings)},
        }},
        DivisionResults: []*pb.RoundResults{{
            {", ".join(f"{{Results: []int32{{{', '.join(map(str, round['results']))}}}}}" for round in division_results)},
        }},
        ClassPrizes:          []int32{{{", ".join(map(str, class_prizes))}}},
        GibsonSpread:         {gibson_spread},
        ControlLossThreshold: {control_loss_threshold},
        HopefulnessThreshold: {hopefulness_threshold},
        AllPlayers:           {all_players},
        ValidPlayers:         {valid_players},
        Rounds:               {rounds},
        PlacePrizes:          {place_prizes},
        DivisionSims:         {division_sims},
        ControlLossSims:      {control_loss_sims},
        UseControlLoss:       {str(use_control_loss).lower()},
        AllowRepeatByes:      {str(allow_repeat_byes).lower()},
        RemovedPlayers:       []int32{{{", ".join(map(str, removed_players))}}},
        Seed:                 {seed},
    }}
    return request
}}
"""

# Print the generated function
print(go_function)
