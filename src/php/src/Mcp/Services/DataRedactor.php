<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

/**
 * Data Redactor - redacts sensitive information from tool call logs.
 *
 * Prevents PII, credentials, and secrets from being stored in tool call
 * logs while maintaining enough context for debugging.
 */
class DataRedactor
{
    /**
     * Keys that should always be fully redacted.
     */
    protected const SENSITIVE_KEYS = [
        'password',
        'passwd',
        'secret',
        'token',
        'api_key',
        'apikey',
        'api-key',
        'auth',
        'authorization',
        'bearer',
        'credential',
        'credentials',
        'private_key',
        'privatekey',
        'access_token',
        'refresh_token',
        'session_token',
        'jwt',
        'ssn',
        'social_security',
        'credit_card',
        'creditcard',
        'card_number',
        'cvv',
        'cvc',
        'pin',
        'routing_number',
        'account_number',
        'bank_account',
    ];

    /**
     * Keys containing PII that should be partially redacted.
     */
    protected const PII_KEYS = [
        'email',
        'phone',
        'telephone',
        'mobile',
        'address',
        'street',
        'postcode',
        'zip',
        'zipcode',
        'date_of_birth',
        'dob',
        'birthdate',
        'national_insurance',
        'ni_number',
        'passport',
        'license',
        'licence',
    ];

    /**
     * Replacement string for fully redacted values.
     */
    protected const REDACTED = '[REDACTED]';

    /**
     * Redact sensitive data from an array recursively.
     */
    public function redact(mixed $data, int $maxDepth = 10): mixed
    {
        if ($maxDepth <= 0) {
            return '[MAX_DEPTH_EXCEEDED]';
        }

        if (is_array($data)) {
            return $this->redactArray($data, $maxDepth - 1);
        }

        if (is_string($data)) {
            return $this->redactString($data);
        }

        return $data;
    }

    /**
     * Redact sensitive values from an array.
     */
    protected function redactArray(array $data, int $maxDepth): array
    {
        $result = [];

        foreach ($data as $key => $value) {
            $lowerKey = strtolower((string) $key);

            // Check for fully sensitive keys
            if ($this->isSensitiveKey($lowerKey)) {
                $result[$key] = self::REDACTED;

                continue;
            }

            // Check for PII keys - partially redact
            if ($this->isPiiKey($lowerKey) && is_string($value)) {
                $result[$key] = $this->partialRedact($value);

                continue;
            }

            // Recurse into nested arrays (with depth guard)
            if (is_array($value)) {
                if ($maxDepth <= 0) {
                    $result[$key] = '[MAX_DEPTH_EXCEEDED]';
                } else {
                    $result[$key] = $this->redactArray($value, $maxDepth - 1);
                }

                continue;
            }

            // Check string values for embedded sensitive patterns
            if (is_string($value)) {
                $result[$key] = $this->redactString($value);

                continue;
            }

            $result[$key] = $value;
        }

        return $result;
    }

    /**
     * Check if a key name indicates sensitive data.
     */
    protected function isSensitiveKey(string $key): bool
    {
        foreach (self::SENSITIVE_KEYS as $sensitiveKey) {
            if (str_contains($key, $sensitiveKey)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Check if a key name indicates PII.
     */
    protected function isPiiKey(string $key): bool
    {
        foreach (self::PII_KEYS as $piiKey) {
            if (str_contains($key, $piiKey)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Redact sensitive patterns from a string value.
     */
    protected function redactString(string $value): string
    {
        // Redact bearer tokens
        $value = preg_replace(
            '/Bearer\s+[A-Za-z0-9\-_\.]+/i',
            'Bearer '.self::REDACTED,
            $value
        ) ?? $value;

        // Redact Basic auth
        $value = preg_replace(
            '/Basic\s+[A-Za-z0-9\+\/=]+/i',
            'Basic '.self::REDACTED,
            $value
        ) ?? $value;

        // Redact common API key patterns (key_xxx, sk_xxx, pk_xxx)
        $value = preg_replace(
            '/\b(sk|pk|key|api|token)_[a-zA-Z0-9]{16,}/i',
            '$1_'.self::REDACTED,
            $value
        ) ?? $value;

        // Redact JWT tokens (xxx.xxx.xxx format with base64)
        $value = preg_replace(
            '/eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*/i',
            self::REDACTED,
            $value
        ) ?? $value;

        // Redact UK National Insurance numbers
        $value = preg_replace(
            '/[A-Z]{2}\s?\d{2}\s?\d{2}\s?\d{2}\s?[A-Z]/i',
            self::REDACTED,
            $value
        ) ?? $value;

        // Redact credit card numbers (basic pattern)
        $value = preg_replace(
            '/\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b/',
            self::REDACTED,
            $value
        ) ?? $value;

        return $value;
    }

    /**
     * Partially redact a value, showing first and last characters.
     */
    protected function partialRedact(string $value): string
    {
        $length = strlen($value);

        if ($length <= 4) {
            return self::REDACTED;
        }

        if ($length <= 8) {
            return substr($value, 0, 2).'***'.substr($value, -1);
        }

        // For longer values, show more context
        $showChars = min(3, (int) floor($length / 4));

        return substr($value, 0, $showChars).'***'.substr($value, -$showChars);
    }

    /**
     * Create a summary of array data without sensitive information.
     *
     * Useful for result_summary where we want structure info without details.
     */
    public function summarize(mixed $data, int $maxDepth = 3): mixed
    {
        if ($maxDepth <= 0) {
            return '[...]';
        }

        if (is_array($data)) {
            $result = [];
            $count = count($data);

            // Limit array size in summary
            $limit = 10;
            $truncated = $count > $limit;
            $items = array_slice($data, 0, $limit, true);

            foreach ($items as $key => $value) {
                $lowerKey = strtolower((string) $key);

                // Fully redact sensitive keys
                if ($this->isSensitiveKey($lowerKey)) {
                    $result[$key] = self::REDACTED;

                    continue;
                }

                // Partially redact PII keys
                if ($this->isPiiKey($lowerKey) && is_string($value)) {
                    $result[$key] = $this->partialRedact($value);

                    continue;
                }

                // Recurse with reduced depth
                $result[$key] = $this->summarize($value, $maxDepth - 1);
            }

            if ($truncated) {
                $result['_truncated'] = '... and '.($count - $limit).' more items';
            }

            return $result;
        }

        if (is_string($data)) {
            // Redact first, then truncate (prevents leaking sensitive patterns)
            $redacted = $this->redactString($data);
            if (strlen($redacted) > 100) {
                return substr($redacted, 0, 97).'...';
            }

            return $redacted;
        }

        return $data;
    }
}
