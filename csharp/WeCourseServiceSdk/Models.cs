namespace WeCourseServiceSdk;

public sealed record CourseTime(int DayOfTheWeek, int TimeOfTheDay);

public sealed record ClassTimeSlot(string Start, string End);

public sealed record Course(
    string CourseID,
    string CourseName,
    string RoomID,
    string RoomName,
    string Weeks,
    IReadOnlyList<CourseTime> CourseTimes);

public sealed record Teacher(
    string CourseID,
    string CourseName,
    string CourseCredit,
    string CourseTeacher);

public sealed record WeekCourse(
    string CourseName,
    string TeacherName,
    string RoomName,
    int DayOfTheWeek,
    string TimeOfTheDay);

public sealed record Semester(
    string SemesterID,
    string Ids,
    bool Current);

public sealed record Identity(
    string Role,
    string RoleName,
    string UserCategoryID);

public sealed record Student(
    string FullName,
    string EnglishName,
    string Sex,
    string StartTime,
    string EndTime,
    string SchoolYear,
    string Type,
    string System,
    string Specialty,
    string Class);

public sealed record Grade(
    string CourseID,
    string CourseName,
    string CourseTerm,
    string CourseCredit,
    string CourseGrade,
    string GradePoint);
