# typed: false
# frozen_string_literal: true

RSpec.describe "POST /@:atname/check", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)

    post("/@#{actor.atname}/check")

    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end

  it "アットネームが存在しないとき、422を返すこと" do
    viewer = FactoryBot.create(:actor)
    sign_in(viewer)

    post("/@unknown/check")

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")
  end

  it "アットネームが正しく、他にチェック可能なおすすめフォロー対象が存在するとき、Turbo Streamレスポンスを返すこと" do
    viewer = FactoryBot.create(:actor)
    actor_a = FactoryBot.create(:actor)
    actor_b = FactoryBot.create(:actor)
    actor_c = FactoryBot.create(:actor)

    sign_in(viewer)

    # フォロー関係を設定してサジェスチョンを作成
    FollowProfileUseCase.new.call(source_profile: viewer.profile_record, target_profile: actor_a.profile_record)
    FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_b.profile_record)
    FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_c.profile_record)
    viewer.profile_record.create_suggested_follows!

    # actor_bをチェック前の状態を確認
    suggested_follow_b = viewer.profile_record.suggested_follow_records.find_by(target_profile_id: actor_b.profile_record.id)
    expect(suggested_follow_b.checked_at).to be_nil

    post("/@#{actor_b.atname}/check")

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")

    # actor_bがチェック済みになっていること
    suggested_follow_b.reload
    expect(suggested_follow_b.checked_at).not_to be_nil

    # まだチェック可能なおすすめフォロー対象が残っていることを確認
    expect(viewer.checkable_suggested_followees.exists?).to be(true)
  end

  it "アットネームが正しく、他にチェック可能なおすすめフォロー対象が存在しないとき、検索ページにリダイレクトすること" do
    viewer = FactoryBot.create(:actor)
    actor_a = FactoryBot.create(:actor)
    actor_b = FactoryBot.create(:actor)

    sign_in(viewer)

    # フォロー関係を設定してサジェスチョンを作成（1つだけ）
    FollowProfileUseCase.new.call(source_profile: viewer.profile_record, target_profile: actor_a.profile_record)
    FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_b.profile_record)
    viewer.profile_record.create_suggested_follows!

    # 唯一のサジェスチョンをチェック
    post("/@#{actor_b.atname}/check")

    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(search_path)

    # チェック可能なおすすめフォロー対象が残っていないことを確認
    expect(viewer.checkable_suggested_followees.exists?).to be(false)
  end
end
