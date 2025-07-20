#!/usr/bin/env python3
"""
Monitoring Demo for Click-Lite Log Analytics

This script demonstrates the monitoring capabilities by:
1. Checking system health
2. Viewing real-time metrics
3. Simulating various load scenarios
4. Triggering and viewing alerts
"""

import requests
import json
import time
import random
import threading
from datetime import datetime
from typing import Dict, List, Any

API_URL = "http://localhost:20002/api/v1"


class MonitoringDemo:
    def __init__(self):
        self.api_url = API_URL
        self.running = True

    def check_health(self) -> Dict[str, Any]:
        """Check system health status"""
        response = requests.get(f"{self.api_url}/monitoring/health")
        return response.json()

    def get_metrics(self) -> List[Dict[str, Any]]:
        """Get current system metrics"""
        response = requests.get(f"{self.api_url}/monitoring/metrics")
        return response.json().get("metrics", [])

    def get_alerts(self) -> List[Dict[str, Any]]:
        """Get active alerts"""
        response = requests.get(f"{self.api_url}/monitoring/alerts/active")
        return response.json().get("alerts", [])

    def generate_normal_load(self, duration: int = 60):
        """Generate normal load for the specified duration"""
        print(f"Generating normal load for {duration} seconds...")
        start_time = time.time()
        
        while time.time() - start_time < duration and self.running:
            # Generate 10-50 logs per second
            batch_size = random.randint(10, 50)
            logs = []
            
            for _ in range(batch_size):
                log_level = random.choices(
                    ["debug", "info", "warning", "error"],
                    weights=[10, 70, 15, 5]
                )[0]
                
                logs.append({
                    "level": log_level,
                    "message": f"Normal operation log - {random.choice(['User login', 'API request', 'Data processed', 'Cache hit', 'Task completed'])}",
                    "service": random.choice(["web", "api", "worker", "cache"]),
                    "timestamp": datetime.utcnow().isoformat() + "Z",
                    "attributes": {
                        "user_id": random.randint(1000, 9999),
                        "duration_ms": random.randint(10, 500),
                        "status": "success" if log_level != "error" else "failed"
                    }
                })
            
            try:
                requests.post(
                    f"{self.api_url}/ingest/logs",
                    json=logs,
                    headers={"Content-Type": "application/json"}
                )
            except:
                pass
            
            # Also execute some queries
            if random.random() < 0.3:  # 30% chance to run a query
                self.execute_query()
            
            time.sleep(1)

    def generate_high_load(self, duration: int = 30):
        """Generate high load to trigger alerts"""
        print(f"Generating HIGH LOAD for {duration} seconds...")
        start_time = time.time()
        
        while time.time() - start_time < duration and self.running:
            # Generate 500-1000 logs per second
            batch_size = random.randint(500, 1000)
            logs = []
            
            for _ in range(batch_size):
                logs.append({
                    "level": random.choice(["info", "warning", "error"]),
                    "message": "High load test log",
                    "service": "load-test",
                    "timestamp": datetime.utcnow().isoformat() + "Z",
                    "attributes": {
                        "load_test": True,
                        "batch_size": batch_size
                    }
                })
            
            try:
                requests.post(
                    f"{self.api_url}/ingest/bulk",
                    json=logs,
                    headers={"Content-Type": "application/json"}
                )
            except:
                pass
            
            time.sleep(0.1)

    def generate_slow_queries(self, count: int = 10):
        """Generate slow queries to trigger performance alerts"""
        print(f"Generating {count} slow queries...")
        
        for i in range(count):
            # Complex aggregation query
            sql = """
            SELECT 
                service,
                level,
                DATE_TRUNC('minute', timestamp) as minute,
                COUNT(*) as count,
                AVG(CAST(json_extract(attributes, '$.duration_ms') as FLOAT)) as avg_duration
            FROM logs
            WHERE timestamp > datetime('now', '-1 hour')
            GROUP BY service, level, minute
            ORDER BY minute DESC, count DESC
            """
            
            try:
                requests.post(
                    f"{self.api_url}/query/execute",
                    json={"sql": sql},
                    headers={"Content-Type": "application/json"}
                )
            except:
                pass
            
            time.sleep(0.5)

    def execute_query(self):
        """Execute a random query"""
        queries = [
            "SELECT COUNT(*) FROM logs WHERE level = 'error'",
            "SELECT service, COUNT(*) as count FROM logs GROUP BY service",
            "SELECT * FROM logs ORDER BY timestamp DESC LIMIT 100",
            "SELECT level, COUNT(*) FROM logs WHERE timestamp > datetime('now', '-5 minutes') GROUP BY level"
        ]
        
        sql = random.choice(queries)
        try:
            requests.post(
                f"{self.api_url}/query/execute",
                json={"sql": sql},
                headers={"Content-Type": "application/json"}
            )
        except:
            pass

    def display_metrics_dashboard(self):
        """Display a simple metrics dashboard in the console"""
        while self.running:
            try:
                # Clear screen (works on Unix-like systems)
                print("\033[2J\033[H")
                
                # Get current data
                health = self.check_health()
                metrics = self.get_metrics()
                alerts = self.get_alerts()
                
                # Display header
                print("=" * 80)
                print("CLICK-LITE MONITORING DASHBOARD".center(80))
                print("=" * 80)
                print()
                
                # Display health status
                status = health.get("status", "unknown")
                status_color = {
                    "ok": "\033[92m",      # Green
                    "degraded": "\033[93m", # Yellow
                    "down": "\033[91m"      # Red
                }.get(status, "\033[0m")
                
                print(f"System Status: {status_color}{status.upper()}\033[0m")
                print(f"Version: {health.get('version', 'unknown')}")
                print(f"Uptime: {self.format_uptime(health.get('uptime_seconds', 0))}")
                print()
                
                # Display key metrics
                print("KEY METRICS:")
                print("-" * 40)
                
                metric_map = {m["name"]: m["value"] for m in metrics}
                
                ingestion_rate = metric_map.get("ingestion_rate_per_second", 0)
                query_rate = metric_map.get("query_rate_per_second", 0)
                total_logs = metric_map.get("total_logs_ingested", 0)
                total_queries = metric_map.get("total_queries_executed", 0)
                
                print(f"Ingestion Rate: {ingestion_rate:.1f} logs/sec")
                print(f"Query Rate: {query_rate:.1f} queries/sec")
                print(f"Total Logs: {total_logs:,}")
                print(f"Total Queries: {total_queries:,}")
                print()
                
                # Display performance metrics
                print("PERFORMANCE:")
                print("-" * 40)
                query_avg = metric_map.get("query_duration_ms_avg", 0)
                query_p99 = metric_map.get("query_duration_ms_p99", 0)
                
                print(f"Avg Query Duration: {query_avg:.1f} ms")
                print(f"P99 Query Duration: {query_p99:.1f} ms")
                print()
                
                # Display active alerts
                print("ACTIVE ALERTS:")
                print("-" * 40)
                if alerts:
                    for alert in alerts[:5]:  # Show max 5 alerts
                        severity_color = {
                            "critical": "\033[91m",  # Red
                            "warning": "\033[93m",   # Yellow
                            "info": "\033[94m"       # Blue
                        }.get(alert["severity"], "\033[0m")
                        
                        print(f"{severity_color}[{alert['severity'].upper()}]\033[0m {alert['name']}: {alert['message']}")
                else:
                    print("\033[92mNo active alerts\033[0m")
                
                print()
                print("Press Ctrl+C to stop monitoring...")
                
            except Exception as e:
                print(f"Error updating dashboard: {e}")
            
            time.sleep(2)

    def format_uptime(self, seconds: float) -> str:
        """Format uptime in human-readable format"""
        days = int(seconds // 86400)
        hours = int((seconds % 86400) // 3600)
        minutes = int((seconds % 3600) // 60)
        return f"{days}d {hours}h {minutes}m"

    def run_demo(self):
        """Run the complete monitoring demo"""
        print("Starting Click-Lite Monitoring Demo...")
        print()
        
        # Start dashboard in a separate thread
        dashboard_thread = threading.Thread(target=self.display_metrics_dashboard)
        dashboard_thread.daemon = True
        dashboard_thread.start()
        
        try:
            # Phase 1: Normal operation
            print("Phase 1: Normal operation (60 seconds)")
            self.generate_normal_load(60)
            
            # Phase 2: High load
            print("\nPhase 2: High load test (30 seconds)")
            self.generate_high_load(30)
            
            # Phase 3: Slow queries
            print("\nPhase 3: Slow query test")
            self.generate_slow_queries(10)
            
            # Phase 4: Return to normal
            print("\nPhase 4: Return to normal operation (30 seconds)")
            self.generate_normal_load(30)
            
            print("\nDemo complete! Check http://localhost:5173/monitoring for the web dashboard.")
            
        except KeyboardInterrupt:
            print("\nDemo stopped by user.")
        finally:
            self.running = False


def main():
    # Check if services are running
    try:
        response = requests.get(f"{API_URL}/monitoring/health/live", timeout=2)
        if response.status_code != 200:
            print("Error: Click-Lite services are not responding properly.")
            print("Please ensure the backend is running on port 20002.")
            return
    except requests.exceptions.RequestException:
        print("Error: Cannot connect to Click-Lite services.")
        print("Please ensure:")
        print("1. Backend is running: cd backend && go run main.go")
        print("2. Frontend is running: cd frontend && pnpm dev")
        return
    
    demo = MonitoringDemo()
    demo.run_demo()


if __name__ == "__main__":
    main()