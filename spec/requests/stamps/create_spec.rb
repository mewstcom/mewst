# typed: false
# frozen_string_literal: true

RSpec.describe "POST /posts/:post_id/stamp", type: :request do
  it "未ログインのとき、ルートパスにリダイレクトすること" do
    oauth_app = FactoryBot.create(:oauth_application, :mewst_web)
    target_actor = FactoryBot.create(:actor)
    target_post = FactoryBot.create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    post("/posts/#{target_post.id}/stamp")
    expect(response).to redirect_to(root_path)
  end

  it "存在しないポストIDを指定したとき、422を返すこと" do
    viewer = FactoryBot.create(:actor)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    post("/posts/invalid_post_id/stamp")
    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.content_type).to eq("text/vnd.turbo-stream.html; charset=utf-8")
  end

  it "正常にスタンプを作成できるとき、200を返すこと" do
    oauth_app = FactoryBot.create(:oauth_application, :mewst_web)
    viewer = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    target_post = FactoryBot.create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    expect(PostRecord.count).to eq(1)
    expect(StampRecord.count).to eq(0)

    post("/posts/#{target_post.id}/stamp")
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to eq("text/vnd.turbo-stream.html; charset=utf-8")

    expect(PostRecord.count).to eq(1)
    expect(StampRecord.count).to eq(1)

    # スタンプが正しく作成されていることを確認
    stamp = StampRecord.last
    expect(stamp.profile_record).to eq(viewer.profile_record)
    expect(stamp.post_record).to eq(target_post)
    expect(stamp.stamped_at).to be_present
  end

  it "すでにスタンプ済みのポストに再度スタンプしたとき、重複作成されずに200を返すこと" do
    oauth_app = FactoryBot.create(:oauth_application, :mewst_web)
    viewer = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    target_post = FactoryBot.create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    # 事前にスタンプを作成
    CreateStampUseCase.new.call(viewer: viewer, target_post: target_post)
    expect(StampRecord.count).to eq(1)

    # 再度スタンプを実行
    post("/posts/#{target_post.id}/stamp")
    expect(response).to have_http_status(:ok)
    expect(response.content_type).to eq("text/vnd.turbo-stream.html; charset=utf-8")

    # スタンプが重複作成されていないことを確認
    expect(StampRecord.count).to eq(1)
  end

  it "削除されたポストにスタンプしようとしたとき、422を返すこと" do
    oauth_app = FactoryBot.create(:oauth_application, :mewst_web)
    viewer = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    target_post = FactoryBot.create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)
    post_id = target_post.id

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    # ポストを削除
    target_post.destroy!

    post("/posts/#{post_id}/stamp")
    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.content_type).to eq("text/vnd.turbo-stream.html; charset=utf-8")
  end

  it "Turbo Streamレスポンスが正しい形式で返されること" do
    oauth_app = FactoryBot.create(:oauth_application, :mewst_web)
    viewer = FactoryBot.create(:actor)
    target_actor = FactoryBot.create(:actor)
    target_post = FactoryBot.create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    post("/posts/#{target_post.id}/stamp")
    expect(response).to have_http_status(:ok)

    # Turbo Stream形式のレスポンスが含まれることを確認
    expect(response.body).to include("<turbo-stream")
    expect(response.body).to include("action=\"update\"")
    expect(response.body).to include("target=\"posts-#{target_post.id}-stamp\"")
  end

  it "エラー時にTurbo Streamでエラーメッセージが返されること" do
    viewer = FactoryBot.create(:actor)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    post("/posts/invalid_post_id/stamp")
    expect(response).to have_http_status(:unprocessable_entity)

    # エラー時のTurbo Stream形式のレスポンスを確認
    expect(response.body).to include("<turbo-stream")
    expect(response.body).to include("action=\"after\"")
    expect(response.body).to include("target=\"posts-invalid_post_id-stamp\"")
    expect(response.body).to include("data-controller=\"flash-toast-dispatch\"")
    expect(response.body).to include("ポストが見つかりませんでした")
  end
end
