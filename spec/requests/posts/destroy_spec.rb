# typed: false
# frozen_string_literal: true

RSpec.describe "DELETE /posts/:post_id", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトされること" do
    actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: actor.profile)

    delete "/posts/#{post.id}"

    expect(response).to have_http_status(:found)
    expect(response).to redirect_to(root_path)
  end

  it "ログインしているとき、自分のポストを削除できること" do
    viewer = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: viewer.profile)

    sign_in viewer
    delete "/posts/#{post.id}"

    expect(response).to have_http_status(:see_other)
    expect(response).to redirect_to(home_path)
    expect(flash[:notice]).to eq("投稿を削除しました")
    expect(Post.kept.exists?(id: post.id)).to be(false)
  end

  it "他のユーザーのポストを削除しようとした場合、バリデーションエラーが発生すること" do
    viewer = FactoryBot.create(:actor)
    other_actor = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: other_actor.profile)

    sign_in viewer
    expect { delete "/posts/#{post.id}" }.to raise_error(ActiveModel::ValidationError)
  end

  it "存在しないポストIDで削除しようとした場合、バリデーションエラーが発生すること" do
    viewer = FactoryBot.create(:actor)

    sign_in viewer
    expect { delete "/posts/nonexistent_id" }.to raise_error(ActiveModel::ValidationError)
  end

  it "既に削除されたポストを削除しようとした場合、バリデーションエラーが発生すること" do
    viewer = FactoryBot.create(:actor)
    post = FactoryBot.create(:post, profile: viewer.profile)
    post.discard!

    sign_in viewer
    expect { delete "/posts/#{post.id}" }.to raise_error(ActiveModel::ValidationError)
  end
end
