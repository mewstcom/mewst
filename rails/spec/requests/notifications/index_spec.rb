# typed: false
# frozen_string_literal: true

RSpec.describe "GET /notifications", type: :request do
  it "認証されていない場合、ルートページにリダイレクトすること" do
    get("/notifications")

    expect(response).to have_http_status(:found)
    expect(response).to redirect_to("/")
  end

  it "認証されている場合、通知一覧ページが正常に表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get("/notifications")

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/html")
  end

  it "認証されている場合、通知がある状態で正常に表示されること" do
    FactoryBot.create(:oauth_application, :mewst_web)
    actor = FactoryBot.create(:actor)
    other_actor = FactoryBot.create(:actor)
    post = CreatePostUseCase.new.call(viewer: actor, content: "Hello", canonical_url: "").post
    CreateStampUseCase.new.call(viewer: other_actor, target_post: post)
    sign_in(actor)

    get("/notifications")

    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/html")
    expect(response.body).to include("notification")
  end

  it "無効なカーソルパラメータの場合、リダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    get("/notifications", params: {after: "invalid_cursor"})

    expect(response).to have_http_status(:moved_permanently)
    expect(response).to redirect_to("/notifications")
  end
end
