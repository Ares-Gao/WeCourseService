package io.github.aresgao.wecourseservice;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.regex.Pattern;

public record WeCourseConfig(
        String SchoolName,
        String MangerType,
        String MangerURL,
        String CalendarFirst,
        int SocketPort,
        String LoginType,
        String AuthServerURL,
        String ServiceURL) {
    public static WeCourseConfig load(String path) throws IOException {
        var json = Files.readString(Path.of(path));
        return new WeCourseConfig(
                stringValue(json, "SchoolName"),
                stringValue(json, "MangerType"),
                stringValue(json, "MangerURL"),
                stringValue(json, "CalendarFirst"),
                Integer.parseInt(stringValue(json, "SocketPort")),
                optionalStringValue(json, "LoginType", "direct"),
                optionalStringValue(json, "AuthServerURL", ""),
                optionalStringValue(json, "ServiceURL", ""));
    }

    public String baseUrl() {
        return MangerURL.replaceAll("/+$", "") + "/";
    }

    private static String stringValue(String json, String key) {
        return optionalStringValue(json, key, null);
    }

    private static String optionalStringValue(String json, String key, String defaultValue) {
        var stringMatcher = Pattern.compile("\"" + key + "\"\\s*:\\s*\"([^\"]*)\"").matcher(json);
        if (stringMatcher.find()) {
            return stringMatcher.group(1);
        }
        var numberMatcher = Pattern.compile("\"" + key + "\"\\s*:\\s*(\\d+)").matcher(json);
        if (numberMatcher.find()) {
            return numberMatcher.group(1);
        }
        if (defaultValue != null) {
            return defaultValue;
        }
        throw new IllegalArgumentException("Missing config key: " + key);
    }
}
