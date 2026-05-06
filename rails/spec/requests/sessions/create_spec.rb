# typed: false
# frozen_string_literal: true

RSpec.describe "POST /sign_in", type: :request do
  it "メールアドレスが空のとき、422を返すこと" do
    post("/sign_in", params: {
      session_form: {
        email: "",
        password: "password"
      }
    })

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "パスワードが空のとき、422を返すこと" do
    post("/sign_in", params: {
      session_form: {
        email: "test@example.com",
        password: ""
      }
    })

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "メールアドレスが無効な形式のとき、422を返すこと" do
    post("/sign_in", params: {
      session_form: {
        email: "invalid_email",
        password: "password"
      }
    })

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "存在しないメールアドレスのとき、422を返すこと" do
    post("/sign_in", params: {
      session_form: {
        email: "nonexistent@example.com",
        password: "password"
      }
    })

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "パスワードが間違っているとき、422を返すこと" do
    user = FactoryBot.create(:user_record, email: "test@example.com", password: "correct_password")
    FactoryBot.create(:actor, user_record: user)

    post("/sign_in", params: {
      session_form: {
        email: "test@example.com",
        password: "wrong_password"
      }
    })

    expect(response).to have_http_status(:unprocessable_entity)
  end

  it "正しいメールアドレスとパスワードのとき、ホーム画面にリダイレクトすること" do
    user = FactoryBot.create(:user_record, email: "test@example.com", password: "password")
    FactoryBot.create(:actor, user_record: user)

    post("/sign_in", params: {
      session_form: {
        email: "test@example.com",
        password: "password"
      }
    })

    expect(response).to redirect_to(home_path)
    expect(flash[:notice]).to eq("ログインしました")
  end

  it "正しいメールアドレスとパスワードのとき、セッションが作成されること" do
    user = FactoryBot.create(:user_record, email: "test@example.com", password: "password")
    actor = FactoryBot.create(:actor, user_record: user)

    expect {
      post("/sign_in", params: {
        session_form: {
          email: "test@example.com",
          password: "password"
        }
      })
    }.to change(SessionRecord, :count).by(1)

    session = SessionRecord.last
    expect(session.actor_record).to eq(actor)
  end

  it "既にログインしているとき、ホーム画面にリダイレクトすること" do
    user = FactoryBot.create(:user_record, email: "test@example.com", password: "password")
    actor = FactoryBot.create(:actor, user_record: user)

    # ログイン状態をシミュレート
    sign_in(actor, password: "password")

    post("/sign_in", params: {
      session_form: {
        email: "test@example.com",
        password: "password"
      }
    })

    expect(response).to redirect_to(home_path)
    expect(flash[:notice]).to eq("すでにログインしています")
  end
end
