/**
 * Species API Client Tests
 * Tests for species exclusion management functionality
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { getExcludedSpecies, isSpeciesExcluded, toggleSpeciesExclusion } from './speciesApi';
import { fetchWithCSRF } from '$lib/utils/api';

// Mock dependencies
vi.mock('$lib/utils/api', () => ({
  fetchWithCSRF: vi.fn(),
}));

vi.mock('$lib/utils/logger', () => ({
  loggers: {
    api: {
      debug: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
    },
  },
}));

describe('speciesApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  describe('getExcludedSpecies', () => {
    it('should fetch and return excluded species list', async () => {
      const mockResponse = {
        excluded: ['American Robin', 'House Sparrow', 'European Starling'],
        count: 3,
      };

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await getExcludedSpecies();

      expect(global.fetch).toHaveBeenCalledWith('/api/v2/settings/species/excluded', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      expect(result).toEqual(mockResponse.excluded);
    });

    it('should return empty array on fetch error', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
      });

      const result = await getExcludedSpecies();

      expect(result).toEqual([]);
    });

    it('should return empty array on network error', async () => {
      global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

      const result = await getExcludedSpecies();

      expect(result).toEqual([]);
    });

    it('should return empty array when response is malformed', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      const result = await getExcludedSpecies();

      expect(result).toEqual([]);
    });
  });

  describe('isSpeciesExcluded', () => {
    it('should return true if species is in excluded list', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          excluded: ['American Robin', 'House Sparrow'],
          count: 2,
        }),
      });

      const result = await isSpeciesExcluded('American Robin');

      expect(result).toBe(true);
    });

    it('should return false if species is not in excluded list', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          excluded: ['American Robin', 'House Sparrow'],
          count: 2,
        }),
      });

      const result = await isSpeciesExcluded('Blue Jay');

      expect(result).toBe(false);
    });

    it('should return false on error', async () => {
      global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

      const result = await isSpeciesExcluded('American Robin');

      expect(result).toBe(false);
    });
  });

  describe('toggleSpeciesExclusion', () => {
    it('should call fetchWithCSRF with correct parameters', async () => {
      vi.mocked(fetchWithCSRF).mockResolvedValue(
        new Response(JSON.stringify({ success: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      );

      await toggleSpeciesExclusion('American Robin');

      expect(fetchWithCSRF).toHaveBeenCalledWith('/api/v2/detections/ignore', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          common_name: 'American Robin',
        }),
      });
    });

    it('should throw error on API failure', async () => {
      const apiError = new Error('API Error');
      vi.mocked(fetchWithCSRF).mockRejectedValue(apiError);

      await expect(toggleSpeciesExclusion('American Robin')).rejects.toThrow('API Error');
    });

    it('should handle species names with special characters', async () => {
      vi.mocked(fetchWithCSRF).mockResolvedValue(
        new Response(JSON.stringify({ success: true }), { status: 200 })
      );

      await toggleSpeciesExclusion("Cooper's Hawk");

      expect(fetchWithCSRF).toHaveBeenCalledWith(
        '/api/v2/detections/ignore',
        expect.objectContaining({
          body: JSON.stringify({
            common_name: "Cooper's Hawk",
          }),
        })
      );
    });
  });
});
