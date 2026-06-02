from prometheus_client import CONTENT_TYPE_LATEST, Counter, Gauge, Histogram, generate_latest


risk_scores_calculated_total = Counter("risk_scores_calculated_total", "Total risk scores calculated.")
risk_breaches_total = Counter("risk_breaches_total", "Total high or critical risk breaches.")
risk_anomalies_detected_total = Counter("risk_anomalies_detected_total", "Total risk anomalies detected.")
risk_recommendations_created_total = Counter("risk_recommendations_created_total", "Total risk recommendations created.")
risk_calculation_duration_seconds = Histogram("risk_calculation_duration_seconds", "Risk calculation duration in seconds.")
risk_score_current = Gauge("risk_score_current", "Most recent calculated risk score.")
var_current = Gauge("var_current", "Most recent calculated Value at Risk.")
drawdown_current = Gauge("drawdown_current", "Most recent calculated max drawdown.")
kafka_publish_errors_total = Counter("kafka_publish_errors_total", "Total Kafka publish errors.")


def metrics_response() -> tuple[bytes, str]:
    return generate_latest(), CONTENT_TYPE_LATEST
