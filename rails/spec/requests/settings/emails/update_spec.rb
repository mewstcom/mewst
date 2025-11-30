# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /settings/email", type: :request do
  it "ログインしていないとき、ルートページにリダイレクトすること" do
    patch "/settings/email", params: {email_update_form: {new_email: "new@example.com"}}

    expect(response).to redirect_to("/")
  end

  it "ログインしているとき、新しいメールアドレスが空の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/email", params: {email_update_form: {new_email: ""}}

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("メールアドレス")
  end

  it "ログインしているとき、新しいメールアドレスが無効な形式の場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/email", params: {email_update_form: {new_email: "invalid-email"}}

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("メールアドレス")
  end

  it "ログインしているとき、新しいメールアドレスが既に使用されている場合、エラーが表示されること" do
    actor = FactoryBot.create(:actor)
    FactoryBot.create(:user_record, email: "taken@example.com")
    sign_in(actor)

    patch "/settings/email", params: {email_update_form: {new_email: "taken@example.com"}}

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("メールアドレス")
  end

  it "ログインしているとき、有効な新しいメールアドレスを送信した場合、確認メールが送信されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    patch "/settings/email", params: {email_update_form: {new_email: "new@example.com"}}

    expect(response).to redirect_to(new_email_confirmation_path)
    expect(flash[:notice]).to eq("確認用のメールを送信しました")
    expect(session[:email_confirmation_id]).to be_present
  end

  it "ログインしているとき、メールアドレス確認用のレコードが作成されること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    expect {
      patch "/settings/email", params: {email_update_form: {new_email: "new@example.com"}}
    }.to change(EmailConfirmationRecord, :count).by(1)

    email_confirmation = EmailConfirmationRecord.last
    expect(email_confirmation.not_nil!.email).to eq("new@example.com")
    expect(email_confirmation.not_nil!.event).to eq("email_update")
  end
end
