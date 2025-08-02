# typed: false
# frozen_string_literal: true

RSpec.describe "GET /followees", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/followees"
    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end

  it "ログインしているとき、フォロー中のユーザー一覧ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/followees"
    expect(response).to have_http_status(:ok)
  end

  it "フォロー中のユーザーがいるとき、フォロー順でユーザーが表示されること" do
    actor = FactoryBot.create(:actor)
    target_actor1 = FactoryBot.create(:actor)
    target_actor2 = FactoryBot.create(:actor)
    sign_in actor

    # フォロー関係を作成（followed_atの時間を違うものにする）
    FollowRecord.create!(
      source_profile_id: actor.profile_record.id,
      target_profile_id: target_actor1.profile_record.id,
      followed_at: 2.days.ago
    )
    FollowRecord.create!(
      source_profile_id: actor.profile_record.id,
      target_profile_id: target_actor2.profile_record.id,
      followed_at: 1.day.ago
    )

    get "/followees"
    expect(response).to have_http_status(:ok)
  end

  it "無効なカーソルが指定されたとき、フォロー一覧ページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/followees", params: {after: "invalid_cursor"}
    expect(response).to have_http_status(:moved_permanently)
    expect(response).to redirect_to(followee_list_path)
  end

  it "ページネーションのパラメータが正常に処理されること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    # 複数のフォロー関係を作成
    16.times do |i|
      target_actor = FactoryBot.create(:actor)
      FollowRecord.create!(
        source_profile_id: actor.profile_record.id,
        target_profile_id: target_actor.profile_record.id,
        followed_at: i.days.ago
      )
    end

    get "/followees"
    expect(response).to have_http_status(:ok)
  end
end
