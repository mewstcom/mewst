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
    Follow.create!(
      source_profile: actor.profile,
      target_profile: target_actor1.profile,
      followed_at: 2.days.ago
    )
    Follow.create!(
      source_profile: actor.profile,
      target_profile: target_actor2.profile,
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
      Follow.create!(
        source_profile: actor.profile,
        target_profile: target_actor.profile,
        followed_at: i.days.ago
      )
    end

    get "/followees"
    expect(response).to have_http_status(:ok)
  end
end
