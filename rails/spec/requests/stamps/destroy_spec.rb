# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /posts/:post_id/stamp", type: :request do
  it "入力データが正しいとき、スタンプを削除して200を返すこと" do
    oauth_app = create(:oauth_application, :mewst_web)
    viewer = create(:actor)
    target_actor = create(:actor)
    post = create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    CreateStampUseCase.new.call(
      viewer:,
      target_post: post
    )

    expect(PostRecord.count).to eq(1)
    expect(StampRecord.count).to eq(1)

    delete("/posts/#{post.id}/stamp")
    expect(response).to have_http_status(:ok)

    expect(PostRecord.count).to eq(1)
    expect(StampRecord.count).to eq(0)
  end

  it "存在しないスタンプを削除しようとしたとき、特にエラーなく完了すること" do
    oauth_app = create(:oauth_application, :mewst_web)
    viewer = create(:actor)
    target_actor = create(:actor)
    post = create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    # ログイン状態にする
    post(sign_in_path, params: {session_form: {email: viewer.email, password: "passw0rd"}})

    # スタンプが存在しない状態でスタンプ削除を実行
    expect(StampRecord.count).to eq(0)

    delete("/posts/#{post.id}/stamp")
    expect(response).to have_http_status(:ok)

    expect(StampRecord.count).to eq(0)
  end

  it "認証されていないとき、ルートページにリダイレクトすること" do
    oauth_app = create(:oauth_application, :mewst_web)
    target_actor = create(:actor)
    post = create(:post_record, profile_record: target_actor.profile_record, oauth_application: oauth_app)

    delete("/posts/#{post.id}/stamp")
    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end
end
