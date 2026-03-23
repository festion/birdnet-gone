"""Tests for VicoHome camera still integration in the display app."""

import json
import os
import tempfile
import pytest


def test_load_camera_detections_returns_dict_keyed_by_lowercase_name():
    """Camera detections are loaded as a dict keyed by lowercase common name."""
    from birdnet_display import load_camera_detections

    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump([
            {"name": "Northern Cardinal", "image_url": "https://example.com/cardinal.jpg",
             "confidence": 95.0, "timestamp": 1742745600, "source": "camera"},
            {"name": "Blue Jay", "image_url": "https://example.com/jay.jpg",
             "confidence": 88.0, "timestamp": 1742745500, "source": "camera"},
        ], f)
        f.flush()
        result = load_camera_detections(f.name)

    os.unlink(f.name)
    assert "northern cardinal" in result
    assert "blue jay" in result
    assert result["northern cardinal"] == "https://example.com/cardinal.jpg"
    assert result["blue jay"] == "https://example.com/jay.jpg"


def test_load_camera_detections_missing_file_returns_empty_dict():
    """Missing file returns empty dict — graceful degradation."""
    from birdnet_display import load_camera_detections
    result = load_camera_detections("/nonexistent/path/detections.json")
    assert result == {}


def test_load_camera_detections_malformed_json_returns_empty_dict():
    """Malformed JSON returns empty dict."""
    from birdnet_display import load_camera_detections

    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        f.write("not valid json{{{")
        f.flush()
        result = load_camera_detections(f.name)

    os.unlink(f.name)
    assert result == {}


def test_load_camera_detections_empty_array_returns_empty_dict():
    """Empty array returns empty dict."""
    from birdnet_display import load_camera_detections

    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump([], f)
        f.flush()
        result = load_camera_detections(f.name)

    os.unlink(f.name)
    assert result == {}


def test_load_camera_detections_skips_entries_without_image_url():
    """Entries missing image_url are skipped."""
    from birdnet_display import load_camera_detections

    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump([
            {"name": "Robin", "image_url": "", "confidence": 90.0},
            {"name": "Sparrow", "image_url": "https://example.com/sparrow.jpg", "confidence": 85.0},
        ], f)
        f.flush()
        result = load_camera_detections(f.name)

    os.unlink(f.name)
    assert "robin" not in result
    assert "sparrow" in result


def test_load_camera_detections_first_entry_wins_on_duplicate_species():
    """When multiple entries exist for the same species, the first (most recent) wins."""
    from birdnet_display import load_camera_detections

    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump([
            {"name": "Cardinal", "image_url": "https://example.com/newest.jpg", "confidence": 95.0},
            {"name": "Cardinal", "image_url": "https://example.com/older.jpg", "confidence": 80.0},
        ], f)
        f.flush()
        result = load_camera_detections(f.name)

    os.unlink(f.name)
    assert result["cardinal"] == "https://example.com/newest.jpg"
