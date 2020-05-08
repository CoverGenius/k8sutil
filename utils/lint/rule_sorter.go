package lint

import (
	"fmt"
)

/* This object is used to store all the rules belonging to a resource group and looks like:

&rulesorter.RuleSorter{
	rules:24:(*lint.Rule)(0xc00039caf0),
	edges:24:map[lint.RuleID]lint.RuleID{}
}
*/
type RuleSorter struct {
	rules map[RuleID]*Rule
	edges map[RuleID]map[RuleID]RuleID
}

/**
* Retrieve the rule given its ID
* May as well implement this since I have to make a map for other operations anyway
**/
func (r *RuleSorter) Get(id RuleID) *Rule {
	return r.rules[id]
}

/**
* Create a new RuleSorter given a list of rules
* Usual use case is to use the RuleSorter to access the rules in the correct order!
**/
func NewRuleSorter(rules []*Rule) *RuleSorter {
	e := make(map[RuleID]map[RuleID]RuleID)
	r := make(map[RuleID]*Rule)
	for _, rule := range rules {
		r[rule.ID] = rule
		e[rule.ID] = make(map[RuleID]RuleID)
		for _, prereq := range rule.Prereqs {
			e[rule.ID][prereq] = prereq
		}
	}
	return &RuleSorter{edges: e, rules: r}
}

func (r *RuleSorter) GetDependentRules(masterID RuleID) []*Rule {
	ruleIDs := r.getDependents(masterID)
	var rules []*Rule
	for _, id := range ruleIDs {
		rules = append(rules, r.rules[id])
	}
	return rules
}

/**
*	Given a rule (identified by its ID), get all the rules that are dependent upon it.
*   This implies that those rules' Condition functions are keeping a reference to the same struct.
* 	Ie, you would never have a rule dependent on another if they are referring to different objects.
**/
func (r *RuleSorter) getDependents(masterID RuleID) []RuleID {
	var dependentIDs []RuleID
	for id := range r.rules {
		for _, masterRuleID := range r.rules[id].Prereqs {
			if masterRuleID == masterID {
				dependentIDs = append(dependentIDs, id)
				dependentIDs = append(dependentIDs, r.getDependents(id)...)
			}
		}
	}
	return dependentIDs
}

/**
* Use this when you want to retrieve AND get rid of all rules that are dependent on a particular rule.
* Usually you want to use this when a rule fails, and you would like to avoid executing
* the rules that depend on this rule's success.
**/
func (r *RuleSorter) PopDependentRules(masterID RuleID) []*Rule {
	dependents := r.GetDependentRules(masterID)
	// now just delete them from the map.
	for _, rule := range dependents {
		delete(r.edges, rule.ID)
	}
	return dependents
}

func (r *RuleSorter) IsEmpty() bool {
	return len(r.edges) == 0
}

/**
* When you need to know which rule you should execute next, call this method. It will remove
* the rule from the data structure and return it.
* The algorithm is as follows:

1. Find a rule with no dependencies, in case of multiple such rules the first one is chosen
2. Find all the rules which depend on this rule, and remove it from it's dependency list
3. Remove the rule itself from the edge map
4. Return the rule
**/
func (r *RuleSorter) PopNextAvailable() *Rule {
	var ruleID RuleID
	var cycle bool
	for id, incoming := range r.edges {
		if incoming == nil || len(incoming) == 0 {
			ruleID = id
			cycle = true
			break
		}
	}
	// If we don't have any empty edges list, that means
	// we have a cycle somewhere
	if !cycle {
		fmt.Printf("%#v\n", r)
		panic("There is a cycle in your rule dependencies, you can't do it like this")
	}
	for _, id := range r.getDependents(ruleID) {
		// update their edges so that they don't remember ruleID anymore!
		delete(r.edges[id], ruleID)
	}
	// now please forget totally about this ruleID from the edges
	delete(r.edges, ruleID)
	// its map is also gone, (it would have been empty anyways)
	return r.rules[ruleID]
}
