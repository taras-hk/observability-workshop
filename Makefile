simulate-v1:
	@echo "Starting V1 Observability simulation"
	@./scripts/simulate_v1_observability.sh

simulate-v2:
	@echo "Starting V2 Observability simulation"
	@./scripts/simulate_v2_observability.sh

simulate-v3:
	@echo "Starting V3 Observability simulation"
	@./scripts/simulate_v3_observability.sh

test-alerts-v1:
	@echo "🔥 Running V1 Alerts Testing"
	@./scripts/test_alerts_v1.sh

test-alerts-v2:
	@echo "⚡ Running V2 Alerts Testing"
	@./scripts/test_alerts_v2.sh

test-alerts-v3:
	@echo "🚀 Running V3 Alerts Testing"
	@./scripts/test_alerts_v3.sh


start-all:
	@docker compose up -d
	@echo "Starting all services..."
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Services are ready!"
	@echo "Jaeger UI: http://localhost:16686"
	@echo "Prometheus: http://localhost:9090"
	@echo "Kibana: http://localhost:5601"

stop-all:
	@docker compose down
	@echo "All services have been stopped"
