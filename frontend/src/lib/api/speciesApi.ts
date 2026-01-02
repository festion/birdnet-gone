/**
 * Species API Client
 * Handles species-related API operations including exclusion management
 */

import { fetchWithCSRF } from '$lib/utils/api';
import { loggers } from '$lib/utils/logger';

const logger = loggers.api;

/**
 * Response from the excluded species endpoint
 */
interface ExcludedSpeciesResponse {
  excluded: string[];
  count: number;
}

/**
 * Get the list of excluded species
 * @returns Promise resolving to array of excluded species common names
 */
export async function getExcludedSpecies(): Promise<string[]> {
  try {
    const response = await fetch('/api/v2/settings/species/excluded', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch excluded species: ${response.statusText}`);
    }

    const data: ExcludedSpeciesResponse = await response.json();
    logger.debug('Fetched excluded species', { count: data.count });

    return data.excluded;
  } catch (error) {
    logger.error('Error fetching excluded species', error as Error);
    return []; // Return empty array on error
  }
}

/**
 * Check if a species is currently excluded
 * @param commonName - Common name of the species to check
 * @returns Promise resolving to true if species is excluded, false otherwise
 */
export async function isSpeciesExcluded(commonName: string): Promise<boolean> {
  const excluded = await getExcludedSpecies();
  return excluded.includes(commonName);
}

/**
 * Toggle species exclusion status
 * This uses the existing /api/v2/detections/ignore endpoint
 * @param commonName - Common name of the species to toggle
 * @returns Promise resolving when toggle is complete
 */
export async function toggleSpeciesExclusion(commonName: string): Promise<void> {
  try {
    logger.debug('Toggling species exclusion', { species: commonName });

    await fetchWithCSRF('/api/v2/detections/ignore', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        common_name: commonName,
      }),
    });

    logger.info('Species exclusion toggled successfully', { species: commonName });
  } catch (error) {
    logger.error('Error toggling species exclusion', error as Error, { species: commonName });
    throw error;
  }
}
