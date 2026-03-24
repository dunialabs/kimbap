/**
 * Simple IP geolocation utility
 * This is a basic implementation for demonstration purposes.
 * In production, you should use a proper IP geolocation service.
 */

interface GeoLocation {
  country: string;
  city: string;
  countryCode: string;
}

// Basic IP range to location mapping for demonstration
const IP_GEO_MAP: Record<string, GeoLocation> = {
  // Google Cloud ranges
  '34.': { country: 'United States', city: 'Google Cloud US', countryCode: 'US' },
  '35.': { country: 'United States', city: 'Google Cloud US', countryCode: 'US' },
  
  // Common IP ranges
  '172.': { country: 'Private Network', city: 'Local Network', countryCode: 'LO' },
  '192.168.': { country: 'Private Network', city: 'Local Network', countryCode: 'LO' },
  '10.': { country: 'Private Network', city: 'Local Network', countryCode: 'LO' },
  
  // Some example public ranges
  '194.50.': { country: 'Netherlands', city: 'Amsterdam', countryCode: 'NL' },
  '198.176.': { country: 'United States', city: 'New York', countryCode: 'US' },
  '103.171.': { country: 'Singapore', city: 'Singapore', countryCode: 'SG' },
  
  // IPv6 mapped IPv4
  '::ffff:172.': { country: 'Private Network', city: 'Local Network', countryCode: 'LO' },
  '::ffff:198.176.': { country: 'United States', city: 'New York', countryCode: 'US' },
  '::ffff:103.171.': { country: 'Singapore', city: 'Singapore', countryCode: 'SG' },
};

/**
 * Get geographic location for an IP address
 * @param ip IP address string
 * @returns GeoLocation object with country, city, and country code
 */
export function getGeoLocationFromIP(ip: string): GeoLocation {
  // Clean up IPv6 mapped IPv4 addresses
  const cleanedIP = ip.replace(/^::ffff:/, '');
  
  // Try to find matching IP range
  for (const [prefix, location] of Object.entries(IP_GEO_MAP)) {
    if (ip.startsWith(prefix) || cleanedIP.startsWith(prefix.replace('::ffff:', ''))) {
      return location;
    }
  }
  
  // Default fallback
  return {
    country: 'Unknown',
    city: 'Unknown Location',
    countryCode: 'XX'
  };
}

/**
 * Aggregate request counts by geographic location
 * @param ipRequestCounts Map of IP addresses to request counts
 * @returns Array of location data with aggregated request counts
 */
export function aggregateLocationStats(ipRequestCounts: Map<string, number>): Array<{
  country: string;
  city: string;
  requests: number;
  percentage: number;
}> {
  const locationCounts = new Map<string, { location: GeoLocation; requests: number }>();
  const totalRequests = Array.from(ipRequestCounts.values()).reduce((sum, count) => sum + count, 0);
  
  // Aggregate by location
  for (const [ip, requests] of ipRequestCounts) {
    const location = getGeoLocationFromIP(ip);
    const locationKey = `${location.country}-${location.city}`;
    
    if (locationCounts.has(locationKey)) {
      locationCounts.get(locationKey)!.requests += requests;
    } else {
      locationCounts.set(locationKey, { location, requests });
    }
  }
  
  // Convert to array and sort by request count
  const result = Array.from(locationCounts.values())
    .map(({ location, requests }) => ({
      country: location.country,
      city: location.city,
      requests,
      percentage: totalRequests > 0 ? (requests / totalRequests) * 100 : 0
    }))
    .sort((a, b) => b.requests - a.requests)
    .slice(0, 5); // Top 5 locations
  
  return result;
}