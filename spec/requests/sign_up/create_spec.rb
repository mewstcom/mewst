# typed: strict
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "POST /sign_up", type: :request do
  it "認証されていない場合、有効なメールアドレスでEmailConfirmationが作成されて確認メール送信画面にリダイレクトされること" do
    post sign_up_path, params: {email_confirmation_form: {email: "test@example.com"}}

    expect(response).to redirect_to(new_email_confirmation_path)
    expect(flash[:notice]).to eq("確認用のメールを送信しました")

    email_confirmation = EmailConfirmation.find_by(email: "test@example.com")
    expect(email_confirmation).not_to be_nil
    expect(email_confirmation.not_nil!.event).to eq("sign_up")
    expect(session[:email_confirmation_id]).to eq(email_confirmation.not_nil!.id)
  end

  it "認証されていない場合、無効なメールアドレスでは422ステータスでフォームが再表示されること" do
    post sign_up_path, params: {email_confirmation_form: {email: "invalid-email"}}

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("メールアドレスは不正な値です")
    expect(EmailConfirmation.find_by(email: "invalid-email")).to be_nil
  end

  it "認証されていない場合、空のメールアドレスでは422ステータスでフォームが再表示されること" do
    post sign_up_path, params: {email_confirmation_form: {email: ""}}

    expect(response).to have_http_status(:unprocessable_entity)
    expect(response.body).to include("メールアドレスを入力してください")
  end

  it "既に認証されている場合、ホームパスにリダイレクトされること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post sign_up_path, params: {email_confirmation_form: {email: "test@example.com"}}

    expect(response).to redirect_to(home_path)
  end

  it "サインアップが停止されている場合、500エラーが発生すること" do
    allow(Rails.configuration.mewst).to receive(:[]).with("sign_up_stopper_enabled").and_return(true)

    expect {
      post sign_up_path, params: {email_confirmation_form: {email: "test@example.com"}}
    }.to raise_error(RuntimeError)
  end

  it "同じメールアドレスで複数回送信した場合、新しいEmailConfirmationが作成されること" do
    post sign_up_path, params: {email_confirmation_form: {email: "test@example.com"}}
    first_confirmation = EmailConfirmation.find_by(email: "test@example.com")

    post sign_up_path, params: {email_confirmation_form: {email: "test@example.com"}}

    expect(response).to redirect_to(new_email_confirmation_path)
    confirmations = EmailConfirmation.where(email: "test@example.com")
    expect(confirmations.count).to eq(2)
    expect(session[:email_confirmation_id]).not_to eq(first_confirmation.not_nil!.id)
  end
end
