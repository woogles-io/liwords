import sys
import re

class Player:
    def __init__(self, name, opponents, scores):
        self.name = name
        self.opponents = opponents
        self.scores = scores

def parse_t_file(file_path):
    players = []
    total_rounds = -1
    player_number = 0
    with open(file_path, 'r') as file:
        for line in file:
            parts = line.split(';')
            player_name_and_opps = re.split(r'\d+', parts[0], 1)
            name_info = player_name_and_opps[0].split(',')
            name_str = f"{name_info[1].strip()} {name_info[0].strip()}"
                
            scores_str = parts[1].strip()
            opponent_strs = player_name_and_opps[1].split()
            opponents = []
            for opponent_str in opponent_strs:
                opponent_number = int(opponent_str)
                if opponent_number == 0:
                    opponents.append(player_number)
                else:
                    opponents.append(int(opponent_str) - 1)
            
            if total_rounds == -1:
                total_rounds = len(opponents)
            elif total_rounds != len(opponents):
                print(f"Error: all players must have the same number of rounds ({total_rounds} != {len(opponents)}): {line}")
                sys.exit(1)

            scores = list(map(int, scores_str.split()))
            if len(scores) != total_rounds:
                print("Error: all players must have the same number of scores")
                sys.exit(1)
            
            players.append(Player(name_str, opponents, scores))
            player_number += 1
    return players, total_rounds

def generate_go_code(players, tournament_name, number_of_rounds, total_rounds):
    go_code = ""

    func_name = f"Create{tournament_name.capitalize()}AfterRound{number_of_rounds}PairRequest"
    go_code += f"func {func_name}() *pb.PairRequest {{\n"
    go_code += "    request := &pb.PairRequest{\n"
    go_code += "        PairMethod:           pb.PairMethod_COP,\n"
    go_code += f"        Players:              {len(players)},\n"
    go_code += f"        Rounds:               {total_rounds},\n"
    go_code += "        PlayerNames:          []string{"
    go_code += ', '.join([f'"{player.name}"' for player in players])
    go_code += "},\n"
    
    pairings = []
    results = []
    
    for player in players:
        pairings.append(player.opponents[:number_of_rounds])
        results.append(player.scores[:number_of_rounds])

    go_code += "        DivisionPairings:     []*pb.RoundPairings{\n"
    for round_pair in zip(*pairings):
        go_code += "            {Pairings: []int32{"
        go_code += ', '.join(map(str, round_pair))
        go_code += "}},\n"
    go_code += "        },\n"

    go_code += "        DivisionResults:      []*pb.RoundResults{\n"
    for round_result in zip(*results):
        go_code += "            {Results: []int32{"
        go_code += ', '.join(map(str, round_result))
        go_code += "}},\n"
    go_code += "        },\n"

    go_code += "        Classes:              []int32{4},\n"
    go_code += "        ClassPrizes:          []int32{2},\n"
    go_code += "        GibsonSpreads:        []int32{300, 250, 200},\n"
    go_code += "        ControlLossThreshold: 0.25,\n"
    go_code += "        HopefulnessThreshold: 0.02,\n"
    go_code += "        PlacePrizes:          2,\n"
    go_code += "        DivisionSims:         1000,\n"
    go_code += "        ControlLossSims:      1000,\n"
    go_code += "        UseControlLoss:       false,\n"
    go_code += "        AllowRepeatByes:      false,\n"
    go_code += "    }\n"
    go_code += "    return request\n"
    go_code += "}\n\n"

    return go_code

def main():
    if len(sys.argv) != 4:
        print("Usage: python script_name.py <path_to_tsh_t_file> <tournament_name> <number_of_rounds>")
        sys.exit(1)

    file_path = sys.argv[1]
    tournament_name = sys.argv[2]
    number_of_rounds = int(sys.argv[3])

    players, total_rounds = parse_t_file(file_path)
    go_code = generate_go_code(players, tournament_name, number_of_rounds, total_rounds)
    print(go_code)

if __name__ == "__main__":
    main()
