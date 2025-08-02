# typed: false
# frozen_string_literal: true

RSpec.describe "GET /search", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    get "/search"

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、検索ページが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/search"

    expect(response).to have_http_status(:ok)
  end

  it "ログインしているとき、提案プロフィールが表示されること" do
    viewer = FactoryBot.create(:actor)
    actor_a = FactoryBot.create(:actor)
    actor_b = FactoryBot.create(:actor)

    sign_in(viewer)

    # フォロー関係を設定してサジェスチョンを作成
    FollowProfileUseCase.new.call(source_profile: viewer.profile_record, target_profile: actor_a.profile_record)
    FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_b.profile_record)
    viewer.profile_record.create_suggested_follows!

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include(actor_b.profile_record.atname)
    expect(response.body).to include("suggested-profiles-#{actor_b.profile_record.atname}")
  end

  it "ログインしているとき、提案プロフィールがない場合、空のメッセージが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("おすすめプロフィールはありません")
  end

  it "ログインしているとき、検索フォームが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("form")
    expect(response.body).to include(search_profile_list_path)
  end

  it "ログインしているとき、フォローボタンが表示されること" do
    viewer = FactoryBot.create(:actor)
    actor_a = FactoryBot.create(:actor)
    actor_b = FactoryBot.create(:actor)

    sign_in(viewer)

    # フォロー関係を設定してサジェスチョンを作成
    FollowProfileUseCase.new.call(source_profile: viewer.profile_record, target_profile: actor_a.profile_record)
    FollowProfileUseCase.new.call(source_profile: actor_a.profile_record, target_profile: actor_b.profile_record)
    viewer.profile_record.create_suggested_follows!

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("suggested-profiles-#{actor_b.profile_record.atname}")
  end

  it "ログインしているとき、複数の提案プロフィールが表示されること" do
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

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include(actor_b.profile_record.atname)
    expect(response.body).to include(actor_c.profile_record.atname)
  end

  it "ログインしているとき、チェック済みの提案プロフィールは表示されないこと" do
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

    # actor_bをチェック済みにする
    suggested_follow_b = viewer.profile_record.suggested_follow_records.find_by(target_profile_id: actor_b.profile_record.id)
    suggested_follow_b.update!(checked_at: Time.current)

    get "/search"

    expect(response).to have_http_status(:ok)
    expect(response.body).not_to include("suggested-profiles-#{actor_b.profile_record.atname}")
    expect(response.body).to include("suggested-profiles-#{actor_c.profile_record.atname}")
  end
end
