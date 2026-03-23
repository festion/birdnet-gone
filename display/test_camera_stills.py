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


def test_camera_image_overrides_thumbnail_and_cache(tmp_path):
    """Camera images take priority over BirdNET thumbnails and local cache."""
    from birdnet_display import load_camera_detections

    # Simulate what get_bird_data() will do: build bird list, then overlay camera images
    birds = [
        {"name": "Northern Cardinal", "scientific_name": "Cardinalis cardinalis",
         "image_url": "https://wikimedia.org/cardinal_stock.jpg", "copyright": "CC BY-SA"},
        {"name": "Blue Jay", "scientific_name": "Cyanocitta cristata",
         "image_url": "https://wikimedia.org/jay_stock.jpg", "copyright": "CC BY"},
        {"name": "House Sparrow", "scientific_name": "Passer domesticus",
         "image_url": "/static/bird_images_cache/sparrow.jpg", "copyright": "John Doe"},
    ]

    # Camera only has Cardinal and Sparrow
    detections_file = tmp_path / "detections.json"
    detections_file.write_text(json.dumps([
        {"name": "Northern Cardinal", "image_url": "https://cdn.vicohome.io/cardinal_real.jpg",
         "confidence": 95.0, "timestamp": 1742745600, "source": "camera"},
        {"name": "House Sparrow", "image_url": "https://cdn.vicohome.io/sparrow_real.jpg",
         "confidence": 88.0, "timestamp": 1742745500, "source": "camera"},
    ]))

    camera_images = load_camera_detections(str(detections_file))

    # Apply camera overlay (same logic as get_bird_data will use)
    for bird in birds:
        camera_url = camera_images.get(bird['name'].lower())
        if camera_url:
            bird['image_url'] = camera_url
            bird['copyright'] = ""

    # Cardinal: replaced with camera still
    assert birds[0]['image_url'] == "https://cdn.vicohome.io/cardinal_real.jpg"
    assert birds[0]['copyright'] == ""

    # Blue Jay: no camera match, keeps original
    assert birds[1]['image_url'] == "https://wikimedia.org/jay_stock.jpg"
    assert birds[1]['copyright'] == "CC BY"

    # Sparrow: replaced with camera still
    assert birds[2]['image_url'] == "https://cdn.vicohome.io/sparrow_real.jpg"
    assert birds[2]['copyright'] == ""
