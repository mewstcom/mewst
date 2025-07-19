# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /password", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    patch "/password", params: {
      password_reset_form: {
        password: "newpassword123"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "すでにサインイン済みの場合、フラッシュメッセージとともにホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: actor.email,
      succeeded_at: Time.current)

    sign_in(actor)
    set_session(email_confirmation_id: email_confirmation.id)

    patch "/password", params: {
      password_reset_form: {
        password: "newpassword123"
      }
    }

    expect(response).to redirect_to(home_path)
  end

  it "メール確認が完了していない場合、ルートパスにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation, code: "123456")

    set_session(email_confirmation_id: email_confirmation.id)

    patch "/password", params: {
      password_reset_form: {
        password: "newpassword123"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "有効なパスワードが渡された場合、パスワードが更新されてサインインページにリダイレクトすること" do
    user = FactoryBot.create(:user, email: "test@example.com", password: "oldpassword")
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      succeeded_at: Time.current)

    set_session(email_confirmation_id: email_confirmation.id)

    patch "/password", params: {
      password_reset_form: {
        password: "newpassword123"
      }
    }

    expect(response).to redirect_to(sign_in_path)
    expect(flash[:notice]).to be_present

    # パスワードが更新されたことを確認
    user.reload
    expect(user.authenticate("newpassword123")).to be_truthy
    expect(user.authenticate("oldpassword")).to be_falsey
  end

  it "無効なパスワードが渡された場合、422ステータスでエラーを返すこと" do
    FactoryBot.create(:user, email: "test@example.com")
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      code: "123456",
      succeeded_at: Time.current)

    set_session(email_confirmation_id: email_confirmation.id)

    patch "/password", params: {
      password_reset_form: {
        password: "" # 空のパスワードでバリデーションエラー
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "成功時にセッションがリセットされること" do
    FactoryBot.create(:user, email: "test@example.com")
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      code: "123456",
      succeeded_at: Time.current)

    set_session(email_confirmation_id: email_confirmation.id)

    patch "/password", params: {
      password_reset_form: {
        password: "newpassword123"
      }
    }

    expect(response).to redirect_to(sign_in_path)

    # セッションがリセットされたことを確認するため、再度アクセスしてリダイレクトを確認
    get "/password/edit"
    expect(response).to redirect_to(root_path)
  end
end
