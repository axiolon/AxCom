// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import "fmt"

// validateAndSort checks the dependency graph of the provided modules
// and returns them in topological (dependency-first) order using Kahn's algorithm.
//
// Errors returned:
//   - A required module is not in the enabled set.
//   - A circular dependency is detected.
func validateAndSort(modules []Module) ([]Module, error) {
	// Build a name -> Module map for quick lookup.
	byName := make(map[string]Module, len(modules))
	for _, m := range modules {
		byName[m.Name()] = m
	}

	// Validate that every declared dependency is present in the enabled set.
	for _, m := range modules {
		for _, dep := range m.Requires() {
			if _, ok := byName[dep]; !ok {
				return nil, fmt.Errorf(
					"module %q requires %q but %q is either disabled or not registered; "+
						"enable it in config.yaml under modules.%s.enabled: true",
					m.Name(), dep, dep, dep,
				)
			}
		}
	}

	// Build in-degree map and adjacency list (dep -> dependents).
	inDegree := make(map[string]int, len(modules))
	dependents := make(map[string][]string, len(modules))

	for _, m := range modules {
		if _, exists := inDegree[m.Name()]; !exists {
			inDegree[m.Name()] = 0
		}
		for _, dep := range m.Requires() {
			inDegree[m.Name()]++
			dependents[dep] = append(dependents[dep], m.Name())
		}
	}

	// Kahn's algorithm: start with nodes that have no dependencies.
	queue := make([]string, 0, len(modules))
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	sorted := make([]Module, 0, len(modules))
	for len(queue) > 0 {
		// Pop from front.
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, byName[current])

		for _, dependent := range dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If not all modules were sorted, there is a cycle.
	if len(sorted) != len(modules) {
		cycle := findCycle(byName, inDegree)
		return nil, fmt.Errorf("circular dependency detected among modules: %s", cycle)
	}

	return sorted, nil
}

// findCycle returns a human-readable description of the cycle for error messages.
func findCycle(byName map[string]Module, inDegree map[string]int) string {
	// Nodes still with inDegree > 0 are part of the cycle.
	// Walk one of them to surface the cycle path.
	var start string
	for name, deg := range inDegree {
		if deg > 0 {
			start = name
			break
		}
	}
	if start == "" {
		return "(unknown)"
	}

	// Build reverse map: module -> its requirements still in the cycle.
	visited := map[string]bool{}
	path := []string{start}
	current := start

	for {
		visited[current] = true
		m := byName[current]
		advanced := false
		for _, dep := range m.Requires() {
			if inDegree[dep] > 0 && !visited[dep] {
				path = append(path, dep)
				current = dep
				advanced = true
				break
			}
		}
		if !advanced {
			// Close the cycle back to a visited node.
			for _, dep := range m.Requires() {
				if inDegree[dep] > 0 {
					path = append(path, dep)
					break
				}
			}
			break
		}
	}

	result := ""
	for i, p := range path {
		if i > 0 {
			result += " -> "
		}
		result += p
	}
	return result
}
