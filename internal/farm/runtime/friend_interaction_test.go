package runtime

import (
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
)

func interactionTestInfo(id int64, interactionType, desc string) *logic.ItemInfo {
	return &logic.ItemInfo{
		ID:              id,
		Type:            interactionItemType,
		CanUse:          1,
		InteractionType: interactionType,
		Desc:            desc,
	}
}

// 自用判定走通用土地元数据 + SELF_USABLE 白名单：5003 闪电变异瓶的描述不含
// "好友/他人"，也能命中自用（对齐 bot isSelfLandInteractionMetadata）。
func TestSelfInteractionPredicateIncludesLightningBottle(t *testing.T) {
	i5003 := interactionTestInfo(5003, "additemuseItem", "雨落成诗活动道具，使未成熟的作物立即发生闪电变异，种子、枯萎阶段和天工作物不可用。")
	if !isSelfLandInteractionInfo(i5003) {
		t.Fatal("5003 应命中自用白名单")
	}
	if !isFriendLandInteractionInfo(interactionTestInfo(301103, "additemuseitem", "")) {
		t.Fatal("301103 应命中好友土地清单")
	}
	// 种草/黄金虫只能作用于好友农场，不在自用白名单
	if isSelfLandInteractionInfo(interactionTestInfo(301101, "additemuseitem", "在好友农场放置可获得30经验")) {
		t.Fatal("301101 不应命中自用白名单")
	}
	// 青蛙使坏瓶随青蛙内容清理移除：真实配置无 interaction_type（农场级协议道具），
	// 删除 FRIEND_FARM_ITEM_IDS 显式集合后不进入任何互动道具清单（与 rust 一致）
	i5005 := interactionTestInfo(5005, "", "雨落成诗活动道具，在好友农场放出一只捣乱的小青蛙，可得30经验。")
	if isFriendLandInteractionInfo(i5005) || isSelfLandInteractionInfo(i5005) {
		t.Fatal("5005 不应再进入任何互动道具清单")
	}
	// 非互动类型直接排除
	if isSelfLandInteractionInfo(interactionTestInfo(5003, "other", "")) {
		t.Fatal("非 additemuseitem 不应命中自用白名单")
	}
}

// 自用清单分类器：自用道具归类 land，好友专属道具被跳过。
func TestSelfInteractionKindClassifier(t *testing.T) {
	i5003 := interactionTestInfo(5003, "additemuseitem", "使未成熟的1*1作物立即发生闪电变异。")
	i301101 := interactionTestInfo(301101, "additemuseitem", "在好友农场放置可获得30经验")

	selfKind := func(info *logic.ItemInfo) string {
		if isSelfLandInteractionInfo(info) {
			return "land"
		}
		return ""
	}
	if kind := selfKind(i5003); kind != "land" {
		t.Fatalf("5003 分类 = %q, 期望 land", kind)
	}
	if kind := selfKind(i301101); kind != "" {
		t.Fatalf("301101 分类 = %q, 期望空(跳过)", kind)
	}
}
