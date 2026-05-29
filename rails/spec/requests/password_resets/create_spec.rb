# typed: false
# frozen_string_literal: true

RSpec.describe "POST /password_reset", type: :request do
  it "ログインしていないとき、有効なメールアドレスでパスワードリセット確認メールが送信されること" do
    post "/password_reset", params: {
      email_confirmation_form: {
        email: "test@example.com"
      }
    }

    expect(response).to redirect_to(new_email_confirmation_path)
    expect(flash[:notice]).to be_present
    expect(session[:email_confirmation_id]).to be_present
  end

  it "ログインしていないとき、無効なメールアドレスで422ステータスが返されること" do
    post "/password_reset", params: {
      email_confirmation_form: {
        email: "invalid-email"
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "ログインしていないとき、メールアドレスが空で422ステータスが返されること" do
    post "/password_reset", params: {
      email_confirmation_form: {
        email: ""
      }
    }

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "ログインしているとき、ホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)
    sign_in(actor)

    post "/password_reset", params: {
      email_confirmation_form: {
        email: "test@example.com"
      }
    }

    expect(response).to redirect_to(home_path)
  end
end
