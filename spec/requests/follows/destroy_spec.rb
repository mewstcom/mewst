# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /@:atname/follow", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    target_actor = FactoryBot.create(:actor)
    delete "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end

  it "存在しないユーザーをアンフォローしようとしたとき、422エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    sign_in actor
    delete "/@nonexistent_user/follow"
    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "自分自身をアンフォローしようとしたとき、422エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    sign_in actor
    delete "/@#{actor.atname}/follow"
    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "正常にアンフォローできること" do
    actor = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    sign_in actor

    # 事前にフォロー関係を作成
    Follow.create!(source_profile: actor.profile, target_profile: target_actor.profile, followed_at: Time.current)
    expect(Follow.count).to eq(1)

    delete "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")

    # フォロー関係が削除されていることを確認
    expect(Follow.count).to eq(0)
  end

  it "フォローしていないユーザーをアンフォローしようとしても正常に処理されること" do
    actor = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    sign_in actor

    expect(Follow.count).to eq(0)

    delete "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")

    expect(Follow.count).to eq(0)
  end
end
