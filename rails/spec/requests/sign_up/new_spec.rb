# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_up", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in actor

    get "/sign_up"

    expect(response).to redirect_to(home_path)
  end

  it "ログインしていないとき、ページが表示されること" do
    get "/sign_up"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("アカウント作成")
  end

  it "ログインしていないとき、メールアドレス入力フォームが表示されること" do
    get "/sign_up"

    expect(response.body).to include("メールアドレス")
    expect(response.body).to include('type="email"')
    expect(response.body).to include('name="email_confirmation_form[email]"')
  end

  it "ログインしていないとき、ログインリンクが表示されること" do
    get "/sign_up"

    expect(response.body).to include("アカウント持ってる？")
  end

  it "サインアップが停止されているとき、フォームが無効化されること" do
    allow(SignUpStopper).to receive(:enabled?).and_return(true)

    get "/sign_up"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include('disabled="disabled"')
  end

  it "サインアップが停止されていないとき、フォームが有効であること" do
    allow(SignUpStopper).to receive(:enabled?).and_return(false)

    get "/sign_up"

    expect(response).to have_http_status(:ok)
    expect(response.body).not_to include('disabled="disabled"')
  end
end
