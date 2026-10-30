"""Storage metric transformations for log storage reports."""

from __future__ import annotations

from typing import Any

BYTES_IN_GB = 1024**3
FILESYSTEM_LABELS = ("fstype", "device", "mountpoint")
VICTORIALOGS_LABELS = ("cluster", "cluster_name", "job", "instance", "path", "namespace", "pod", "service")
VICTORIALOGS_DATA_TYPES = ("storage/big", "storage/small", "storage/inmemory")


def bytes_to_gb(value: float | None) -> float | None:
    if value is None:
        return None
    return round(value / BYTES_IN_GB, 6)


def calculate_filesystem_usage(results: dict[str, Any]) -> list[dict[str, Any]] | dict[str, str]:
    metric_names = (
        "node_filesystem_size_bytes",
        "node_filesystem_avail_bytes",
        "node_filesystem_free_bytes",
    )
    if error := metric_errors(results, metric_names):
        return {"error": error}
    indexed: dict[str, dict[tuple[tuple[str, str], ...], dict[str, Any]]] = {}
    for metric_name in metric_names:
        samples = results.get(metric_name)
        if not isinstance(samples, list):
            return []
        indexed[metric_name] = {
            tuple(
                sorted(
                    (label, value)
                    for label, value in sample.get("metric", {}).items()
                    if label != "__name__"
                )
            ): sample
            for sample in samples
        }
    rows: list[dict[str, Any]] = []
    for labels, size_sample in indexed["node_filesystem_size_bytes"].items():
        available_sample = indexed["node_filesystem_avail_bytes"].get(labels)
        free_sample = indexed["node_filesystem_free_bytes"].get(labels)
        if not available_sample:
            continue
        size = float(size_sample["value"][1])
        available = float(available_sample["value"][1])
        free = float(free_sample["value"][1]) if free_sample else None
        used = size - available
        rows.append(
            {
                "labels": storage_labels(labels),
                "size_gb": bytes_to_gb(size),
                "available_gb": bytes_to_gb(available),
                "free_gb": bytes_to_gb(free),
                "used_gb": bytes_to_gb(used),
                "used_percent": (used / size * 100) if size else None,
            }
        )
    return rows


def storage_labels(labels: tuple[tuple[str, str], ...]) -> dict[str, str]:
    return selected_labels(labels, FILESYSTEM_LABELS)


def calculate_victorialogs_disk_usage(results: dict[str, Any]) -> list[dict[str, Any]] | dict[str, str]:
    if error := metric_errors(results, ("vl_total_disk_space_bytes", "vl_free_disk_space_bytes")):
        return {"error": error}
    total_samples = index_samples_by_labels(results.get("vl_total_disk_space_bytes"))
    free_samples = index_samples_by_labels(results.get("vl_free_disk_space_bytes"))
    rows: list[dict[str, Any]] = []
    for labels, total_sample in total_samples.items():
        free_sample = free_samples.get(labels)
        if not free_sample:
            continue
        total = float(total_sample["value"][1])
        free = float(free_sample["value"][1])
        used = total - free
        rows.append(
            {
                "labels": selected_labels(labels, VICTORIALOGS_LABELS),
                "total_gb": bytes_to_gb(total),
                "free_gb": bytes_to_gb(free),
                "used_gb": bytes_to_gb(used),
                "used_percent": (used / total * 100) if total else None,
            }
        )
    return rows


def calculate_victorialogs_data_size(results: dict[str, Any]) -> list[dict[str, Any]] | dict[str, str]:
    metric_names = (
        "vl_uncompressed_data_size_total",
        "vl_compressed_data_size_total",
        "vl_uncompressed_data_size_by_type",
        "vl_compressed_data_size_by_type",
    )
    if error := metric_errors(results, metric_names):
        return {"error": error}
    uncompressed_total = sample_value(results.get("vl_uncompressed_data_size_total"))
    compressed_total = sample_value(results.get("vl_compressed_data_size_total"))
    uncompressed_by_type = samples_by_type(results.get("vl_uncompressed_data_size_by_type"))
    compressed_by_type = samples_by_type(results.get("vl_compressed_data_size_by_type"))

    rows = [
        data_size_row("total", uncompressed_total, compressed_total)
    ]
    for data_type in VICTORIALOGS_DATA_TYPES:
        rows.append(
            data_size_row(
                data_type,
                uncompressed_by_type.get(data_type),
                compressed_by_type.get(data_type),
            )
        )
    extra_types = sorted((set(uncompressed_by_type) | set(compressed_by_type)) - set(VICTORIALOGS_DATA_TYPES))
    for data_type in extra_types:
        rows.append(data_size_row(data_type, uncompressed_by_type.get(data_type), compressed_by_type.get(data_type)))
    return rows


def metric_errors(results: dict[str, Any], metric_names: tuple[str, ...]) -> str:
    errors = [
        f"{metric_name}: {value['error']}"
        for metric_name in metric_names
        if isinstance((value := results.get(metric_name)), dict) and "error" in value
    ]
    return "\n".join(errors)


def data_size_row(data_type: str, uncompressed: float | None, compressed: float | None) -> dict[str, Any]:
    return {
        "type": data_type,
        "uncompressed_data_size_gb": bytes_to_gb(uncompressed),
        "compressed_data_size_gb": bytes_to_gb(compressed),
        "compression_ratio": round(uncompressed / compressed, 3) if uncompressed and compressed else None,
    }


def sample_value(samples: Any) -> float | None:
    if not isinstance(samples, list) or not samples:
        return None
    try:
        return float(samples[0]["value"][1])
    except (KeyError, IndexError, TypeError, ValueError):
        return None


def samples_by_type(samples: Any) -> dict[str, float]:
    if not isinstance(samples, list):
        return {}
    values: dict[str, float] = {}
    for sample in samples:
        data_type = sample.get("metric", {}).get("type")
        if not data_type:
            continue
        try:
            values[data_type] = float(sample["value"][1])
        except (KeyError, IndexError, TypeError, ValueError):
            continue
    return values


def selected_labels(labels: tuple[tuple[str, str], ...], allowed_labels: tuple[str, ...]) -> dict[str, str]:
    values = dict(labels)
    return {
        label: values[label]
        for label in allowed_labels
        if label in values
    }


def index_samples_by_labels(samples: Any) -> dict[tuple[tuple[str, str], ...], dict[str, Any]]:
    if not isinstance(samples, list):
        return {}
    return {
        tuple(
            sorted(
                (label, value)
                for label, value in sample.get("metric", {}).items()
                if label != "__name__"
            )
        ): sample
        for sample in samples
    }
