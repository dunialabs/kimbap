import React from 'react';

/**
 * Extract URLs from text and convert to clickable links
 * Returns a React element with clickable links
 */
export function renderErrorMessageWithLinks(message: string): React.ReactNode {
  // URL regex pattern
  const urlRegex = /(https?:\/\/[^\s]+)/g;
  const parts: (string | React.ReactElement)[] = [];
  let lastIndex = 0;
  let match;
  let key = 0;

  while ((match = urlRegex.exec(message)) !== null) {
    // Add text before the URL
    if (match.index > lastIndex) {
      parts.push(message.substring(lastIndex, match.index));
    }

    // Add clickable link
    const url = match[0];
    parts.push(
      <a
        key={key++}
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 underline cursor-pointer font-medium"
        onClick={(e) => {
          e.stopPropagation();
          window.open(url, '_blank', 'noopener,noreferrer');
        }}
      >
        {url}
      </a>
    );

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < message.length) {
    parts.push(message.substring(lastIndex));
  }

  // If no URLs found, return original message
  if (parts.length === 0) {
    return message;
  }

  return <>{parts}</>;
}

/**
 * Check if a message contains URLs
 */
export function hasUrls(message: string): boolean {
  const urlRegex = /https?:\/\/[^\s]+/g;
  return urlRegex.test(message);
}

