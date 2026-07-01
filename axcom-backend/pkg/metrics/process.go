// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"math"
	"os"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	gopsprocess "github.com/shirou/gopsutil/v4/process"
)

// ---------------------------------------------------------------------------
// Runtime + process resource collector
// ---------------------------------------------------------------------------

// RuntimeCollector implements prometheus.Collector and exposes Go runtime
// memory statistics and OS-level process resource usage (CPU %, RSS, VMS)
// all under the ecom_engine namespace.
//
// Register once at startup:
//
//	prometheus.MustRegister(metrics.NewRuntimeCollector())
type RuntimeCollector struct {
	proc *gopsprocess.Process // nil if unavailable; metrics are omitted gracefully

	// Go runtime — memory
	heapAllocBytes *prometheus.Desc
	heapSysBytes   *prometheus.Desc
	heapObjects    *prometheus.Desc
	stackSysBytes  *prometheus.Desc
	nextGCBytes    *prometheus.Desc

	// Go runtime — GC
	gcCyclesTotal       *prometheus.Desc
	gcPauseSecondsTotal *prometheus.Desc

	// Go runtime — concurrency
	goroutines *prometheus.Desc

	// OS process
	cpuPercent *prometheus.Desc
	memRSS     *prometheus.Desc
	memVMS     *prometheus.Desc
}

// NewRuntimeCollector creates a RuntimeCollector. It attempts to attach to the
// current process via gopsutil; if that fails the OS-level metrics are omitted
// at collection time rather than panicking.
func NewRuntimeCollector() *RuntimeCollector {
	fqn := func(sub, name string) string {
		return prometheus.BuildFQName(ns, sub, name)
	}

	var proc *gopsprocess.Process
	pid := os.Getpid()
	if pid >= 0 && pid <= math.MaxInt32 {
		proc, _ = gopsprocess.NewProcess(int32(pid))
	}

	return &RuntimeCollector{
		proc: proc,

		heapAllocBytes: prometheus.NewDesc(
			fqn("runtime", "heap_alloc_bytes"),
			"Bytes of heap objects currently allocated (live + not yet freed).", nil, nil,
		),
		heapSysBytes: prometheus.NewDesc(
			fqn("runtime", "heap_sys_bytes"),
			"Bytes of heap memory obtained from the OS.", nil, nil,
		),
		heapObjects: prometheus.NewDesc(
			fqn("runtime", "heap_objects"),
			"Number of allocated heap objects.", nil, nil,
		),
		stackSysBytes: prometheus.NewDesc(
			fqn("runtime", "stack_sys_bytes"),
			"Bytes of stack memory obtained from the OS.", nil, nil,
		),
		nextGCBytes: prometheus.NewDesc(
			fqn("runtime", "next_gc_bytes"),
			"Target heap size at which the next GC cycle will be triggered.", nil, nil,
		),
		gcCyclesTotal: prometheus.NewDesc(
			fqn("runtime", "gc_cycles_total"),
			"Cumulative number of completed GC cycles.", nil, nil,
		),
		gcPauseSecondsTotal: prometheus.NewDesc(
			fqn("runtime", "gc_pause_seconds_total"),
			"Cumulative time spent in GC stop-the-world pauses, in seconds.", nil, nil,
		),
		goroutines: prometheus.NewDesc(
			fqn("runtime", "goroutines"),
			"Current number of live goroutines.", nil, nil,
		),
		cpuPercent: prometheus.NewDesc(
			fqn("process", "cpu_percent"),
			"Process CPU usage as a percentage of one core (0–100 per core).", nil, nil,
		),
		memRSS: prometheus.NewDesc(
			fqn("process", "memory_rss_bytes"),
			"Resident set size: physical memory pages currently mapped by the process.", nil, nil,
		),
		memVMS: prometheus.NewDesc(
			fqn("process", "memory_vms_bytes"),
			"Virtual memory size: total virtual address space reserved by the process.", nil, nil,
		),
	}
}

// Describe sends all metric descriptors to the Prometheus registry.
func (c *RuntimeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.heapAllocBytes
	ch <- c.heapSysBytes
	ch <- c.heapObjects
	ch <- c.stackSysBytes
	ch <- c.nextGCBytes
	ch <- c.gcCyclesTotal
	ch <- c.gcPauseSecondsTotal
	ch <- c.goroutines
	ch <- c.cpuPercent
	ch <- c.memRSS
	ch <- c.memVMS
}

// Collect reads live runtime/process stats and emits current metric values.
func (c *RuntimeCollector) Collect(ch chan<- prometheus.Metric) {
	// ---- Go runtime --------------------------------------------------------
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	ch <- prometheus.MustNewConstMetric(c.heapAllocBytes, prometheus.GaugeValue, float64(ms.HeapAlloc))
	ch <- prometheus.MustNewConstMetric(c.heapSysBytes, prometheus.GaugeValue, float64(ms.HeapSys))
	ch <- prometheus.MustNewConstMetric(c.heapObjects, prometheus.GaugeValue, float64(ms.HeapObjects))
	ch <- prometheus.MustNewConstMetric(c.stackSysBytes, prometheus.GaugeValue, float64(ms.StackSys))
	ch <- prometheus.MustNewConstMetric(c.nextGCBytes, prometheus.GaugeValue, float64(ms.NextGC))
	ch <- prometheus.MustNewConstMetric(c.gcCyclesTotal, prometheus.CounterValue, float64(ms.NumGC))
	// PauseTotalNs is cumulative nanoseconds; convert to seconds.
	ch <- prometheus.MustNewConstMetric(c.gcPauseSecondsTotal, prometheus.CounterValue, float64(ms.PauseTotalNs)/1e9)
	ch <- prometheus.MustNewConstMetric(c.goroutines, prometheus.GaugeValue, float64(runtime.NumGoroutine()))

	// ---- OS process --------------------------------------------------------
	if c.proc == nil {
		return
	}

	if cpu, err := c.proc.CPUPercent(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.cpuPercent, prometheus.GaugeValue, cpu)
	}

	if mem, err := c.proc.MemoryInfo(); err == nil && mem != nil {
		ch <- prometheus.MustNewConstMetric(c.memRSS, prometheus.GaugeValue, float64(mem.RSS))
		ch <- prometheus.MustNewConstMetric(c.memVMS, prometheus.GaugeValue, float64(mem.VMS))
	}
}
