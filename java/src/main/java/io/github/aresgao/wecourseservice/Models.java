package io.github.aresgao.wecourseservice;

import java.util.List;

record CourseTime(int DayOfTheWeek, int TimeOfTheDay) {
}

record Course(String CourseID, String CourseName, String RoomID, String RoomName, String Weeks, List<CourseTime> CourseTimes) {
}

record Teacher(String CourseID, String CourseName, String CourseCredit, String CourseTeacher) {
}
