/**
 * Media API Types
 * Types for species images and media endpoints
 */

/**
 * Species image response from API
 */
export interface SpeciesImageResponse {
  /** URL to the species image */
  url: string;

  /** License information (optional) */
  licenseInfo?: {
    /** License name (e.g., "CC BY-SA 4.0") */
    name: string;

    /** URL to license details */
    url: string;

    /** Author/photographer name */
    author: string;

    /** URL to author's page */
    authorUrl: string;
  };
}

/**
 * Image provider types supported by the backend
 */
export type ImageProvider = 'auto' | 'wikimedia' | 'avicommons';

/**
 * Fallback policy for image providers
 */
export type FallbackPolicy = 'none' | 'all';
