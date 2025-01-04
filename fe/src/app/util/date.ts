export function now(): string {
  const now = new Date();

  // 获取年月日时分秒
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0'); // 月份从 0 开始，需要 +1
  const day = String(now.getDate()).padStart(2, '0');
  const hours = String(now.getHours()).padStart(2, '0');
  const minutes = String(now.getMinutes()).padStart(2, '0');
  const seconds = String(now.getSeconds()).padStart(2, '0');

  // 格式化时间为 YYYY-MM-DD HH:mm:ss
  const formattedTime = `${year}${month}${day}_${hours}${minutes}${seconds}`;
  console.log(formattedTime); // 输出：2023-10-05 14:30:45
  return formattedTime;
}
