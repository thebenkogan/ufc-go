import { useSchedule } from "./api";
import FullscreenText from "./components/FullscreenText";
import ScheduleTable from "./components/ScheduleTable";

function Schedule() {
	const { data: schedule } = useSchedule();

	if (!schedule) {
		return <FullscreenText text="Loading..." />;
	}

	const displayedSchedule = schedule
		.filter((e) => e.date.slice(5) > new Date().toISOString().slice(5))
		.sort((a, b) => a.date.localeCompare(b.date));

	return (
		<div className="flex flex-col bg-red items-center h-screen">
			<h1 className="text-xl font-bold my-2">Event Schedule</h1>
			<ScheduleTable schedule={displayedSchedule} />
		</div>
	);
}

export default Schedule;
