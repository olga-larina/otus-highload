package zabbix

import (
	"context"
	"fmt"
	"strconv"
	"time"

	zbx "github.com/blacked/go-zabbix"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/olga-larina/otus-highload/pkg/logger"
)

type ZabbixObserver struct {
	zabbixHost   string
	zabbixPort   int
	zabbixPeriod time.Duration
	zabbixName   string
	done         chan struct{}
}

func NewZabbixObserver(zabbixHost string, zabbixPort int, zabbixPeriod time.Duration, zabbixName string) *ZabbixObserver {
	return &ZabbixObserver{
		zabbixHost:   zabbixHost,
		zabbixPort:   zabbixPort,
		zabbixPeriod: zabbixPeriod,
		zabbixName:   zabbixName,
		done:         make(chan struct{}),
	}
}

func (s *ZabbixObserver) Start(ctx context.Context) error {
	if s.zabbixHost != "" && s.zabbixPort != 0 {
		logger.Info(ctx, "starting zabbixObserver")
		s.observeMetrics(ctx)
		logger.Info(ctx, "started zabbixObserver")
	}
	return nil
}

func (s *ZabbixObserver) Stop(ctx context.Context) error {
	if s.zabbixHost != "" && s.zabbixPort != 0 {
		logger.Info(ctx, "stopping zabbixObserver")
		<-ctx.Done()
		<-s.done
		logger.Info(ctx, "stopped zabbixObserver")
	}
	return nil
}

func (s *ZabbixObserver) observeMetrics(ctx context.Context) {
	sender := zbx.NewSender(s.zabbixHost, s.zabbixPort)

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.zabbixPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pkg, err := metrics(s.zabbixName)
				if err != nil {
					logger.Error(ctx, err, "failed to collect metrics")
					continue
				}
				if _, err = sender.Send(pkg); err != nil {
					logger.Error(ctx, err, "failed to send metrics to zabbix")

				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func metrics(host string) (*zbx.Packet, error) {
	mem, err := memory.Get()
	if err != nil {
		return nil, err
	}

	before, err := cpu.Get()
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second * 1)
	after, err := cpu.Get()
	if err != nil {
		return nil, err
	}
	total := float64(after.Total - before.Total)

	var metrics = []*zbx.Metric{
		zbx.NewMetric(host, "mem-used", strconv.FormatUint(mem.Used, 10), time.Now().Unix()),
		zbx.NewMetric(host, "mem-cached", strconv.FormatUint(mem.Cached, 10), time.Now().Unix()),
		zbx.NewMetric(host, "mem-free", strconv.FormatUint(mem.Free, 10), time.Now().Unix()),
		zbx.NewMetric(host, "cpu-user", fmt.Sprintf("%f", float64(after.User-before.User)/total*100), time.Now().Unix()),
		zbx.NewMetric(host, "cpu-system", fmt.Sprintf("%f", float64(after.System-before.System)/total*100), time.Now().Unix()),
		zbx.NewMetric(host, "cpu-idle", fmt.Sprintf("%f", float64(after.Idle-before.Idle)/total*100), time.Now().Unix()),
	}

	return zbx.NewPacket(metrics), nil
}
