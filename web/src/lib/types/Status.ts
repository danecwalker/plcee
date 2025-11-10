export interface Status {
  level: number;
  message: string;
}

export const STATUS_OK: Status = {
  level: 0,
  message: "System OK",
};