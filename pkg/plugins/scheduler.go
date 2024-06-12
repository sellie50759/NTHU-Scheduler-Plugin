package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type CustomSchedulerArgs struct {
	Mode string `json:"mode"`
}

type CustomScheduler struct {
	handle    framework.Handle
	scoreMode string
}

var _ framework.PreFilterPlugin = &CustomScheduler{}
var _ framework.ScorePlugin = &CustomScheduler{}

// Name is the name of the plugin used in Registry and configurations.
const (
	Name              string = "CustomScheduler"
	groupNameLabel    string = "podGroup"
	minAvailableLabel string = "minAvailable"
	leastMode         string = "Least"
	mostMode          string = "Most"
)

func (cs *CustomScheduler) Name() string {
	return Name
}

// New initializes and returns a new CustomScheduler plugin.
func New(obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	cs := CustomScheduler{}
	mode := leastMode
	if obj != nil {
		args := obj.(*runtime.Unknown)
		var csArgs CustomSchedulerArgs
		if err := json.Unmarshal(args.Raw, &csArgs); err != nil {
			fmt.Printf("Error unmarshal: %v\n", err)
		}
		mode = csArgs.Mode
		if mode != leastMode && mode != mostMode {
			return nil, fmt.Errorf("invalid mode, got %s", mode)
		}
	}
	cs.handle = h
	cs.scoreMode = mode
	log.Printf("Custom scheduler runs with the mode: %s.", mode)

	return &cs, nil
}

// filter the pod if the pod in group is less than minAvailable
func (cs *CustomScheduler) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	log.Printf("Pod %s is in Prefilter phase.", pod.Name)
	newStatus := framework.NewStatus(framework.Success, "")

	// TODO
	// 1. extract the label of the pod
	// 2. retrieve the pod with the same group label
	// 3. justify if the pod can be scheduled
	group_label := pod.Labels[groupNameLabel]
	log.Printf("Pod label: %s", group_label)

	labelSet := map[string]string{groupNameLabel: group_label}
	selector := labels.SelectorFromSet(labelSet)

	pods, _ := cs.handle.SharedInformerFactory().Core().V1().Pods().Lister().List(selector)
	log.Printf("Pods len: %d", len(pods))

	min_count, err := strconv.Atoi(pod.Labels[minAvailableLabel])
	if err != nil {
		newStatus := framework.NewStatus(framework.Error, "strconv.Atoi(pod.Labels[minAvailableLabel]) error")
		return nil, newStatus
	}

	if len(pods) >= min_count {
		return nil, newStatus
	} else {
		newStatus = framework.NewStatus(framework.Unschedulable, "")
		return &framework.PreFilterResult{NodeNames: sets.NewString()}, newStatus
	}
}

// PreFilterExtensions returns a PreFilterExtensions interface if the plugin implements one.
func (cs *CustomScheduler) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Score invoked at the score extension point.
func (cs *CustomScheduler) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	log.Printf("Pod %s is in Score phase. Calculate the score of Node %s.", pod.Name, nodeName)
	newStatus := framework.NewStatus(framework.Success, "")
	// TODO
	// 1. retrieve the node allocatable memory
	// 2. return the score based on the scheduler mode
	nodeInfo, err := cs.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		newStatus := framework.NewStatus(framework.Error, err.Error())
		return 0, newStatus
	}
	allocatableMemory := nodeInfo.Allocatable.Memory
	requestedMemory := nodeInfo.Requested.Memory
	memory := allocatableMemory - requestedMemory

	log.Printf("Node %s allocatable memory: %d, requested memory: %d, memory: %d", nodeName, allocatableMemory, requestedMemory, memory)

	if cs.scoreMode == leastMode {
		return -memory, newStatus
	} else {
		return memory, newStatus
	}
}

// ensure the scores are within the valid range
func (cs *CustomScheduler) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	// TODO
	// find the range of the current score and map to the valid range
	if len(scores) == 0 {
		return framework.NewStatus(framework.Error, "no scores to normalize")
	}
	newStatus := framework.NewStatus(framework.Success, "")

	// 找到分数的最小值和最大值
	minScore := scores[0].Score
	maxScore := scores[0].Score
	for _, score := range scores {
		log.Printf("Pod %s. Node %s's socre %d", pod.Name, score.Name, score.Score)
		if score.Score < minScore {
			minScore = score.Score
		}
		if score.Score > maxScore {
			maxScore = score.Score
		}
	}

	// 避免最大值和最小值相同的情况（防止除以零）
	if maxScore == minScore {
		for i := range scores {
			scores[i].Score = 0
		}
		return newStatus
	}

	for i := range scores {
		scores[i].Score = framework.MinNodeScore + (scores[i].Score-minScore)*(framework.MaxNodeScore-framework.MinNodeScore)/(maxScore-minScore)
	}

	return newStatus
}

// ScoreExtensions of the Score plugin.
func (cs *CustomScheduler) ScoreExtensions() framework.ScoreExtensions {
	return cs
}
