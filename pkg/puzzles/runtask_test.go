package puzzles

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/matryer/is"
)

func TestPuzzleGenRunTaskInput(t *testing.T) {
	is := is.New(t)
	in := puzzleGenRunTaskInput(`{"lexicon":"CSW21"}`, ECSTask{
		Cluster:        "woogles",
		TaskDefinition: "puzzle-generator-fargate",
		Subnets:        []string{"subnet-a"},
		SecurityGroups: []string{"sg-a"},
	})
	is.Equal(in.LaunchType, types.LaunchTypeFargate)
	vpc := in.NetworkConfiguration.AwsvpcConfiguration
	is.Equal(vpc.Subnets, []string{"subnet-a"})
	is.Equal(vpc.SecurityGroups, []string{"sg-a"})
	is.Equal(vpc.AssignPublicIp, types.AssignPublicIpEnabled)
	is.Equal(aws.ToString(in.Cluster), "woogles")
	is.Equal(aws.ToString(in.TaskDefinition), "puzzle-generator-fargate")
	is.Equal(aws.ToString(in.Overrides.ContainerOverrides[0].Name), "liwords-puzzlegen")
	is.Equal(in.Overrides.ContainerOverrides[0].Command, []string{"/opt/puzzle-generator", `{"lexicon":"CSW21"}`})
}
