using System.Text.Json;

namespace WeCourseServiceSdk;

public sealed record WeCourseConfig(
    string SchoolName,
    string MangerType,
    string MangerURL,
    string CalendarFirst,
    int SocketPort,
    string LoginType = "direct",
    string AuthServerURL = "",
    string ServiceURL = "",
    string CalendarTimezone = "Asia/Shanghai",
    string CalendarName = "微课表",
    IReadOnlyList<ClassTimeSlot>? ClassTimeSlots = null)
{
    public static WeCourseConfig Load(string path = "../config.json")
    {
        var json = File.ReadAllText(path);
        return JsonSerializer.Deserialize<WeCourseConfig>(json)
            ?? throw new InvalidOperationException("Invalid config file.");
    }

    public string BaseUrl => MangerURL.TrimEnd('/') + "/";
}
