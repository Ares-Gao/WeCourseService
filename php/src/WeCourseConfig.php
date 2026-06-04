<?php

final class WeCourseConfig
{
    public function __construct(
        public readonly string $SchoolName,
        public readonly string $MangerType,
        public readonly string $MangerURL,
        public readonly string $CalendarFirst,
        public readonly int $SocketPort,
        public readonly string $LoginType = 'direct',
        public readonly string $AuthServerURL = '',
        public readonly string $ServiceURL = '',
        public readonly string $CalendarTimezone = 'Asia/Shanghai',
        public readonly string $CalendarName = '微课表',
        public readonly array $ClassTimeSlots = [],
    ) {
    }

    public static function load(string $path = __DIR__ . '/../../config.json'): self
    {
        $data = json_decode(file_get_contents($path), true, 512, JSON_THROW_ON_ERROR);
        return new self(
            $data['SchoolName'],
            $data['MangerType'],
            $data['MangerURL'],
            $data['CalendarFirst'],
            (int) $data['SocketPort'],
            $data['LoginType'] ?? 'direct',
            $data['AuthServerURL'] ?? '',
            $data['ServiceURL'] ?? '',
            $data['CalendarTimezone'] ?? 'Asia/Shanghai',
            $data['CalendarName'] ?? '微课表',
            $data['ClassTimeSlots'] ?? [],
        );
    }

    public function baseUrl(): string
    {
        return rtrim($this->MangerURL, '/') . '/';
    }
}
