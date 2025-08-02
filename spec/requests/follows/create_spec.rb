# typed: false
# frozen_string_literal: true

RSpec.describe "POST /@:atname/follow", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    target_actor = FactoryBot.create(:actor)
    post "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end

  it "存在しないユーザーをフォローしようとしたとき、422エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    sign_in actor
    post "/@nonexistent_user/follow"
    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "自分自身をフォローしようとしたとき、422エラーが発生すること" do
    actor = FactoryBot.create(:actor)
    sign_in actor
    post "/@#{actor.atname}/follow"
    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "正常にフォローできること" do
    actor = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    sign_in actor

    expect(FollowRecord.count).to eq(0)

    post "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")

    # フォロー関係が作成されていることを確認
    expect(FollowRecord.count).to eq(1)
    follow = FollowRecord.first
    expect(follow.source_profile_record).to eq(actor.profile_record)
    expect(follow.target_profile_record).to eq(target_actor.profile_record)
  end

  it "既にフォローしているユーザーをフォローしようとしても正常に処理されること" do
    actor = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    sign_in actor

    # 事前にフォロー関係を作成
    FollowRecord.create!(source_profile_id: actor.profile_record.id, target_profile_id: target_actor.profile_record.id, followed_at: Time.current)
    expect(FollowRecord.count).to eq(1)

    post "/@#{target_actor.atname}/follow"
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to include("text/vnd.turbo-stream.html")

    # フォロー関係が重複して作成されないことを確認
    expect(FollowRecord.count).to eq(1)
  end
end
