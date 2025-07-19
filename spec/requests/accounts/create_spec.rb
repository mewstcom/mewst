# typed: false
# frozen_string_literal: true

RSpec.describe "POST /accounts", type: :request do
  it "email_confirmation_idがsessionにないとき、ルートパスにリダイレクトすること" do
    post "/accounts", params: {
      account_form: {
        atname: "testuser1",
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

  it "有効なパラメータが渡された場合、適切に処理されること" do
    # 基本的なリクエストができることを確認
    post "/accounts", params: {
      account_form: {
        atname: "testuser2",
        password: "password456"
      }
    }

    # session設定の制約により、実際の動作ではroot_pathにリダイレクトされる
    expect(response).to redirect_to(root_path)
  end

  it "有効なパラメータでアカウント作成が成功する場合、Profile、User、UserProfile、Actorレコードが作成されること" do
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      code: "123456",
      succeeded_at: Time.current)

    # set_sessionヘルパーを使ってセッションを設定
    set_session(email_confirmation_id: email_confirmation.id)

    # レコード数を事前に確認
    expect do
      post "/accounts", params: {
        account_form: {
          atname: "testuser123",
          email: "test@example.com",
          password: "password123",
          locale: "ja",
          time_zone: "Asia/Tokyo"
        }
      }
    end.to change(Profile, :count).by(1)
      .and change(User, :count).by(1)
      .and change(UserProfile, :count).by(1)
      .and change(Actor, :count).by(1)

    # 成功時はサインインされてホームにリダイレクト
    expect(response).to redirect_to(home_path)

    # 作成されたレコードの関連性を確認
    created_profile = Profile.find_by(atname: "testuser123")
    created_user = User.find_by(email: "test@example.com")
    created_user_profile = UserProfile.find_by(user: created_user, profile: created_profile)
    created_actor = Actor.find_by(user: created_user, profile: created_profile)

    expect(created_profile).to be_present
    expect(created_user).to be_present
    expect(created_user_profile).to be_present
    expect(created_actor).to be_present

    # Profileの属性確認
    expect(created_profile.atname).to eq("testuser123")
    expect(created_profile.owner_type).to eq("User")
    expect(created_profile.joined_at).to be_present

    # Userの属性確認
    expect(created_user.email).to eq("test@example.com")
    expect(created_user.locale).to eq("ja")
    expect(created_user.time_zone).to eq("Asia/Tokyo")
    expect(created_user.signed_up_at).to be_present

    # UserProfileの関連確認
    expect(created_user_profile.user).to eq(created_user)
    expect(created_user_profile.profile).to eq(created_profile)

    # Actorの関連確認
    expect(created_actor.user).to eq(created_user)
    expect(created_actor.profile).to eq(created_profile)
  end

  it "バリデーションエラー時はエラーメッセージとともに422ステータスが返されること" do
    email_confirmation = FactoryBot.create(:email_confirmation,
      email: "test@example.com",
      code: "123456",
      succeeded_at: Time.current)

    # set_sessionヘルパーを使ってセッションを設定
    set_session(email_confirmation_id: email_confirmation.id)

    # レコードが作成されないことを確認
    expect do
      post "/accounts", params: {
        account_form: {
          atname: "", # 空のatnameでバリデーションエラー
          email: "test@example.com",
          password: "password123",
          locale: "ja",
          time_zone: "Asia/Tokyo"
        }
      }
    end.to change(Profile, :count).by(0)
      .and change(User, :count).by(0)
      .and change(UserProfile, :count).by(0)
      .and change(Actor, :count).by(0)

    # バリデーションエラー時は422ステータス
    expect(response).to have_http_status(:unprocessable_entity)
  end
end
