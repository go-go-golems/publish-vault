import { renderCell, rowKey } from "../../../widgets/cellRenderers";
import type { DataTableWidgetProps, JsonObject } from "../../../widgets/ir";
import { defineWidget } from "../../../widgets/registry";
import { DataTable } from "./DataTable";

export const dataTableWidget = defineWidget<DataTableWidgetProps>({
  type: "DataTable",
  module: "widget.dsl",
  render: (props, _children, ctx) => {
    const rowSelectAction = props.onRowSelect;
    return (
      <DataTable<JsonObject>
        className={props.className}
        rows={props.rows}
        columns={props.columns.map(column => ({
          id: column.id,
          header: ctx.renderValue(column.header),
          align: column.align,
          cell: row =>
            renderCell(
              column.cell,
              row,
              ctx.renderNode,
              (action, context) => ctx.dispatchAction(action, context),
              props.getRowKey
            ),
        }))}
        getRowKey={row => rowKey(row, props.getRowKey)}
        selectedKey={props.selectedKey == null ? null : String(props.selectedKey)}
        emptyMessage={props.emptyMessage}
        onRowSelect={
          rowSelectAction
            ? row =>
                ctx.dispatchAction(rowSelectAction, {
                  row,
                  rowKey: rowKey(row, props.getRowKey),
                  componentType: "DataTable",
                })
            : undefined
        }
      />
    );
  },
});
