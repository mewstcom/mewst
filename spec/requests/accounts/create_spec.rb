# typed: false
# frozen_string_literal: true

RSpec.describe "POST /accounts", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    post "/accounts", params: {
      account_form: {
        atname: "testuser",
        password: "password123"
      }
    }

    expect(response).to redirect_to(root_path)
  end

  it "すでにサインイン済みの場合、フラッシュメッセージとともにホームページにリダイレクトすること" do
    actor = FactoryBot.create(:actor)

    sign_in(actor)

    post "/accounts", params: {
      account_form: {
        atname: "testuser",
        password: "password123"
      }
    }

    expect(response).to redirect_to(home_path)
  end

  it "メール確認が完了していない場合、ルートパスにリダイレクトすること" do
    email_confirmation = FactoryBot.create(:email_confirmation, code: "123456")

    # Set session and make request
    get "/accounts/new"
    session[:email_confirmation_id] = email_confirmation.id

    post "/accounts", params: {
      account_form: {
        atname: "testuser",
        password: "password123"
      }
    }

    expect(response).to redirect_to(root_path)
  end
end
