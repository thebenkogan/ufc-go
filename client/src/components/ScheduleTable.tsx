import { AgGridReact } from "ag-grid-react";
import type { GridOptions } from "ag-grid-community";
import "ag-grid-community/styles/ag-grid.css";
import "ag-grid-community/styles/ag-theme-quartz.css";
import type { EventInfo } from "../types";
import { useNavigate } from "react-router-dom";

interface ScheduleTableProps {
	schedule: EventInfo[];
}

function ScheduleTable({ schedule }: ScheduleTableProps) {
	const navigate = useNavigate();

	const gridOptions: GridOptions<EventInfo> = {
		domLayout: "autoHeight",
		rowData: schedule,
		columnDefs: [
			{ field: "name", headerName: "Event", flex: 1.5 },
			{
				field: "date",
				headerName: "Date",
				valueFormatter: (p) =>
					new Date(p.value).toLocaleDateString("en-US", {
						month: "long",
						day: "numeric",
						timeZone: "UTC",
					}),
				flex: 1,
			},
		],
		onRowClicked: (event) =>
			event.data?.id && navigate(`/?id=${event.data.id}`),
	};

	return (
		<div className="ag-theme-quartz h-3/4 w-3/4">
			<AgGridReact gridOptions={gridOptions} />
		</div>
	);
}

export default ScheduleTable;
