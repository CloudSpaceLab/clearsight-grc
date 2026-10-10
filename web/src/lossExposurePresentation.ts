import type { LossSummary } from "./lossTypes";
import type { RankedBarItem } from "./components/ui/RankedBarList";
import { formatLossMoneyExact } from "./components/losses/lossPresentation";

type Totals = { gross: bigint; recovered: bigint; net: bigint; count: number };

export type LossCurrencyExposure = {
  currency: string;
  count: number;
  rows: RankedBarItem[];
};

export function summarizeLoadedLossExposure(items: readonly LossSummary[], visibleCurrencies = 3) {
  const totals = new Map<string, Totals>();
  let excluded = 0;
  let voided = 0;

  for (const item of items) {
    if (item.loss.status === "VOIDED") {
      voided++;
      continue;
    }
    const { gross_amount_minor: gross, recovered_amount_minor: recovered, net_loss_minor: net, currency } = item.totals;
    const code = currency?.trim().toUpperCase() ?? "";
    const valid = /^[A-Z]{3}$/.test(code)
      && code === item.loss.currency?.trim().toUpperCase()
      && [gross, recovered, net].every((value) => Number.isSafeInteger(value) && value >= 0)
      && gross === recovered + net;
    if (!valid) {
      excluded++;
      continue;
    }
    const group = totals.get(code) ?? { gross: 0n, recovered: 0n, net: 0n, count: 0 };
    group.gross += BigInt(gross);
    group.recovered += BigInt(recovered);
    group.net += BigInt(net);
    group.count++;
    totals.set(code, group);
  }

  // Currency order is determined by record count, never cross-currency monetary value.
  const groups = [...totals.entries()].sort((a, b) => b[1].count - a[1].count || a[0].localeCompare(b[0]));
  const displayed = groups.slice(0, Math.max(0, visibleCurrencies)).map(([currency, amount]): LossCurrencyExposure => {
    const scale = (value: bigint) => amount.gross === 0n ? 0 : Number((value * 1000n) / amount.gross);
    const rows: RankedBarItem[] = [
      { id: "gross", label: "Gross", value: scale(amount.gross), displayValue: formatLossMoneyExact(amount.gross, currency) },
      { id: "net", label: "Net outstanding", value: scale(amount.net), displayValue: formatLossMoneyExact(amount.net, currency) },
      { id: "recovered", label: "Recovered", value: scale(amount.recovered), displayValue: formatLossMoneyExact(amount.recovered, currency) },
    ];
    return { currency, count: amount.count, rows };
  });

  return { groups: displayed, moreCurrencies: groups.length - displayed.length, excluded, voided };
}
