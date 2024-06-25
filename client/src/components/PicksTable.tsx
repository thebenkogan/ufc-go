import { AgGridReact } from "ag-grid-react";
import type { GridOptions } from "ag-grid-community";
import "ag-grid-community/styles/ag-grid.css";
import "ag-grid-community/styles/ag-theme-quartz.css";
import type { PicksWithEvent } from "../types";

interface PicksTableProps {
	picks: PicksWithEvent[];
}

function PicksTable({ picks }: PicksTableProps) {
	console.log(picks);
	const gridOptions: GridOptions<PicksWithEvent> = {
		domLayout: "autoHeight",
		autoSizeStrategy: { type: "fitCellContents", colIds: ["event.name"] },
		rowData: picks,
		columnDefs: [
			{ field: "event.name", headerName: "Event" },
			{
				field: "event.start_time",
				headerName: "Date",
				valueFormatter: (p) =>
					p.value === "LIVE" ? "LIVE" : new Date(p.value).toLocaleString(),
			},
			{
				field: "score",
				headerName: "Score",
				valueFormatter: (p) => p.value || "N/A",
			},
			{
				field: "winners",
				headerName: "Picks",
				valueFormatter: (p) => p.value.join(", "),
				flex: 1,
			},
		],
	};

	return (
		<div className="ag-theme-quartz h-3/4 w-3/4">
			<AgGridReact gridOptions={gridOptions} />
		</div>
	);
}

export default PicksTable;
