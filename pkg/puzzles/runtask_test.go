package puzzles

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/matryer/is"
)

func TestPuzzleGenRunTaskInputEC2(t *testing.T) {
	is := is.New(t)
	in := puzzleGenRunTaskInput(`{"lexicon":"CSW21"}`, ECSTask{Cluster: "woogles-prod", TaskDefinition: "puzzle-generator"})
	is.Equal(aws.ToString(in.Cluster), "woogles-prod")
	is.Equal(aws.ToString(in.TaskDefinition), "puzzle-generator")
	is.Equal(in.LaunchType, types.LaunchType(""))
	is.True(in.NetworkConfiguration == nil)
	is.Equal(in.Overrides.ContainerOverrides[0].Command, []string{"/opt/puzzle-generator", `{"lexicon":"CSW21"}`})
}

func TestPuzzleGenRunTaskInputFargate(t *testing.T) {
	is := is.New(t)
	in := puzzleGenRunTaskInput("{}", ECSTask{
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
	is.Equal(aws.ToString(in.Overrides.ContainerOverrides[0].Name), "liwords-puzzlegen")
}
