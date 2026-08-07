// Package farm is the QQ Farm bot engine.
//
// Subpackages:
//   - tsdk: wazero TSDK loader
//   - protocol: WSS gatepb client (+ LandsNotify dispatch)
//   - runtime: AccountManager + Session (farm/friend/daily ticks)
//   - hub: in-memory status broadcast + run-log ring for UI WebSocket
//   - game: Plant / Item / Shop / Mall / Friend / Visit / Task / Activity RPCs
//   - logic: land analysis, planting strategy, analytics rankings, gameconfig
//   - stats: daily operation counters
//   - push: webhook offline/error notifications
//   - activitycenter: constellation catalog helpers
//   - proto/*: hand-written protowire codecs (plantpb, itempb, friendpb, …)
package farm
