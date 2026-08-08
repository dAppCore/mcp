<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

/**
 * Redact sensitive values from MCP payloads before they are logged or shown.
 *
 * @example
 * $redactor = new DataRedactor();
 * $safe = $redactor->redact(['token' => 'sk_live_secret']);
 */
final class DataRedactor
{
    protected const REDACTED = '[REDACTED]';

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
     * Recursively redact secrets and personal data from a value.
     *
     * @example
     * $safe = $redactor->redact(['authorization' => 'Bearer sk_live_secret']);
     */
    public function redact(mixed $data, int $maxDepth = 10): mixed
    {
        if ($maxDepth <= 0) {
            return '[MAX_DEPTH_EXCEEDED]';
        }

        if (is_object($data)) {
            return $this->redactObject($data, $maxDepth - 1);
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
     * Produce a shortened, redacted preview of a value for dashboards.
     *
     * @example
     * $preview = $redactor->summarize(['email' => 'agent@example.com', 'notes' => 'Long body text']);
     */
    public function summarize(mixed $data, int $maxDepth = 3): mixed
    {
        if ($maxDepth <= 0) {
            return '[...]';
        }

        if (is_object($data)) {
            return $this->summarizeObject($data, $maxDepth - 1);
        }

        if (is_array($data)) {
            $result = [];
            $count = count($data);
            $limit = 10;
            $items = array_slice($data, 0, $limit, true);

            foreach ($items as $key => $value) {
                $lowerKey = strtolower((string) $key);

                if ($this->isSensitiveKey($lowerKey)) {
                    $result[$key] = self::REDACTED;

                    continue;
                }

                if ($this->isPiiKey($lowerKey) && is_string($value)) {
                    $result[$key] = $this->partialRedact($value);

                    continue;
                }

                $result[$key] = $this->summarize($value, $maxDepth - 1);
            }

            if ($count > $limit) {
                $result['_truncated'] = sprintf('... and %d more items', $count - $limit);
            }

            return $result;
        }

        if (is_string($data)) {
            $redacted = $this->redactString($data);

            return strlen($redacted) > 100
                ? substr($redacted, 0, 97).'...'
                : $redacted;
        }

        return $data;
    }

    /**
     * Redact a structured object after normalising it to array-like data.
     *
     * @example
     * $safe = $this->redactObject((object) ['token' => 'sk_live_secret'], 3);
     */
    protected function redactObject(object $data, int $maxDepth): mixed
    {
        $normalised = $this->normaliseObject($data);

        if (is_object($normalised)) {
            return $this->redactObject($normalised, $maxDepth);
        }

        if (is_array($normalised)) {
            return $this->redactArray($normalised, $maxDepth);
        }

        return is_string($normalised)
            ? $this->redactString($normalised)
            : $normalised;
    }

    /**
     * Summarise a structured object after normalising it to array-like data.
     *
     * @example
     * $summary = $this->summarizeObject((object) ['email' => 'agent@example.com'], 2);
     */
    protected function summarizeObject(object $data, int $maxDepth): mixed
    {
        $normalised = $this->normaliseObject($data);

        if (is_object($normalised)) {
            return $this->summarizeObject($normalised, $maxDepth);
        }

        return $this->summarize($normalised, $maxDepth);
    }

    /**
     * Redact one associative array while preserving non-sensitive keys.
     *
     * @example
     * $safe = $this->redactArray(['password' => 'secret', 'status' => 'ok'], 2);
     */
    protected function redactArray(array $data, int $maxDepth): array
    {
        $result = [];

        foreach ($data as $key => $value) {
            $lowerKey = strtolower((string) $key);

            if ($this->isSensitiveKey($lowerKey)) {
                $result[$key] = self::REDACTED;

                continue;
            }

            if ($this->isPiiKey($lowerKey) && is_string($value)) {
                $result[$key] = $this->partialRedact($value);

                continue;
            }

            if (is_array($value)) {
                $result[$key] = $maxDepth <= 0
                    ? '[MAX_DEPTH_EXCEEDED]'
                    : $this->redactArray($value, $maxDepth - 1);

                continue;
            }

            $result[$key] = is_string($value)
                ? $this->redactString($value)
                : $value;
        }

        return $result;
    }

    /**
     * Decide whether an array key should always be fully redacted.
     *
     * @example
     * $sensitive = $this->isSensitiveKey('api_key');
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
     * Decide whether an array key should receive partial personal-data redaction.
     *
     * @example
     * $pii = $this->isPiiKey('email');
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
     * Scrub sensitive token formats from a free-form string.
     *
     * @example
     * $safe = $this->redactString('Bearer sk_live_secret');
     */
    protected function redactString(string $value): string
    {
        $value = preg_replace('/Bearer\s+[A-Za-z0-9\-_\.]+/i', 'Bearer '.self::REDACTED, $value) ?? $value;
        $value = preg_replace('/Basic\s+[A-Za-z0-9+\/=]+/i', 'Basic '.self::REDACTED, $value) ?? $value;
        $value = preg_replace('/\b(sk|pk|key|api|token)_[A-Za-z0-9]{16,}\b/i', '$1_'.self::REDACTED, $value) ?? $value;
        $value = preg_replace('/eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*/i', self::REDACTED, $value) ?? $value;
        $value = preg_replace('/[A-Z]{2}\s?\d{2}\s?\d{2}\s?\d{2}\s?[A-Z]/i', self::REDACTED, $value) ?? $value;
        $value = preg_replace('/\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b/', self::REDACTED, $value) ?? $value;

        return $value;
    }

    /**
     * Partially mask a personally identifiable string while leaving a hint.
     *
     * @example
     * $masked = $this->partialRedact('agent@example.com');
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

        $visible = min(3, (int) floor($length / 4));

        return substr($value, 0, $visible).'***'.substr($value, -$visible);
    }

    /**
     * Convert an object into data that can be traversed by the redactor.
     *
     * @example
     * $normalised = $this->normaliseObject((object) ['token' => 'sk_live_secret']);
     */
    protected function normaliseObject(object $data): mixed
    {
        if ($data instanceof \JsonSerializable) {
            return $data->jsonSerialize();
        }

        return get_object_vars($data);
    }
}
