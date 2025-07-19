# typed: false
# frozen_string_literal: true

RSpec.describe "GET /password/edit", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    get "/password/edit"

    expect(response).to redirect_to(root_path)
  end

  it "すでにサインイン済みの場合、フラッシュメッセージとともにホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: actor.email,
      succeeded_at: Time.current)

    sign_in(actor)
    set_session(email_confirmation_id: email_confirmation.id)

    get "/password/edit"

    expect(response).to redirect_to(home_path)
  end

  it "メール確認が完了していない場合、ルートパスにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation, code: "123456")

    set_session(email_confirmation_id: email_confirmation.id)

    get "/password/edit"

    expect(response).to redirect_to(root_path)
  end

  it "有効なemail_confirmationがあり未認証の場合、パスワードリセットフォームが表示されること" do
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      succeeded_at: Time.current)

    set_session(email_confirmation_id: email_confirmation.id)

    get "/password/edit"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("パスワード")
  end

  it "フォームが正しく表示されること" do
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      succeeded_at: Time.current)

    set_session(email_confirmation_id: email_confirmation.id)

    get "/password/edit"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("password_reset_form")
  end
end
