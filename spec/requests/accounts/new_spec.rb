# typed: false
# frozen_string_literal: true

RSpec.describe "GET /accounts/new", type: :request do
  it "ログインしているとき、フラッシュメッセージとともにホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)

    sign_in(actor)

    get "/accounts/new"

    expect(response).to redirect_to(home_path)
    expect(flash[:notice]).to be_present
  end

  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    get "/accounts/new"

    expect(response).to redirect_to(root_path)
  end

  it "メール確認が完了していない場合、ルートパスにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation)

    # set_sessionヘルパーを使ってセッションを設定
    set_session(email_confirmation_id: email_confirmation.id)

    get "/accounts/new"

    expect(response).to redirect_to(root_path)
  end

  it "メール確認が完了している場合、ページが正常に表示されること" do
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      succeeded_at: Time.current)

    # set_sessionヘルパーを使ってセッションを設定
    set_session(email_confirmation_id: email_confirmation.id)

    get "/accounts/new"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("test@example.com")
  end
end
