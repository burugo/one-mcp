import { useNavigate } from 'react-router-dom';
import { LoginFormCommon } from '@/components/ui/LoginFormCommon';

const Login: React.FC = () => {
    const navigate = useNavigate();
    // 登录成功后跳转首页
    const handlePageSuccess = () => {
        navigate('/');
    };
    return (
        <div className="min-h-screen flex items-center justify-center bg-background">
            <div className="w-full max-w-md p-8">
                <LoginFormCommon onSuccess={handlePageSuccess} />
            </div>
        </div>
    );
};

export default Login; 