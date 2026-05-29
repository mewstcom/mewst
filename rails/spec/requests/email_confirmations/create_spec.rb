# typed: false
# frozen_string_literal: true

RSpec.describe "POST /email_confirmations", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "123456"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "すでにサインイン済みでもemail_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "123456"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "無効な確認コードが送信された場合、422ステータスが返されること" do
    email_confirmation = FactoryBot.create(:email_confirmation_record, code: "123456")
    set_session(email_confirmation_id: email_confirmation.id)

    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "invalid_code"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "有効な確認コードでサインアップイベントの場合、アカウント作成ページにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation_record,
      code: "123456",
      event: "sign_up",
      succeeded_at: nil)
    set_session(email_confirmation_id: email_confirmation.id)

    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "123456"
      }
    }

    expect(email_confirmation.reload.succeeded_at).to be_present
    expect(response).to redirect_to(new_account_path)
  end

  it "有効な確認コードでメールアドレス更新イベントの場合、メール設定ページにリダイレクトすること" do
    user = FactoryBot.create(:user_record)
    actor = FactoryBot.create(:actor, user_record: user)
    sign_in(actor)

    email_confirmation = FactoryBot.create(:email_confirmation_record,
      code: "123456",
      event: "email_update",
      succeeded_at: nil)
    set_session(email_confirmation_id: email_confirmation.id)

    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "123456"
      }
    }

    expect(email_confirmation.reload.succeeded_at).to be_present
    expect(response).to redirect_to(settings_email_path)
    expect(flash[:notice]).to be_present
  end

  it "有効な確認コードでパスワードリセットイベントの場合、パスワード編集ページにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation_record,
      code: "123456",
      event: "password_reset",
      succeeded_at: nil)
    set_session(email_confirmation_id: email_confirmation.id)

    post "/email_confirmations", params: {
      email_confirmation_challenge_form: {
        confirmation_code: "123456"
      }
    }

    expect(email_confirmation.reload.succeeded_at).to be_present
    expect(response).to redirect_to(edit_password_path)
  end
end
